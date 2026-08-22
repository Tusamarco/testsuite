// Package modules contains all test modules for the testsuite.
//
// connectionpool.go implements ConnectionPoolTest, which measures how efficiently
// MySQL handles incoming connections under load. It observes three independent
// layers simultaneously:
//
//  1. The application-side sql.DB pool (Go's database/sql): how long goroutines
//     wait to acquire a pooled connection, and how quickly they release it.
//
//  2. The MySQL server thread cache: whether the server can reuse OS threads for
//     new connections or must spawn new ones, which is orders of magnitude more
//     expensive.
//
//  3. The server's performance_schema socket wait instrumentation: the server's
//     own view of how long it took to accept each new client connection.
//
// By comparing these three layers across different concurrency levels, payload
// sizes, connection lifetimes, and pool sizes, you can pinpoint exactly where
// connection latency is coming from — the Go pool, the OS network stack, or the
// MySQL server — and tune accordingly.
package modules

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"testsuite/internal/config"

	log "github.com/sirupsen/logrus"
)

// cpScratch is the name of the temporary table created when --writeSize > 0.
// Using UPDATE on a pre-seeded row (rather than INSERT) avoids auto_increment
// contention between workers and keeps the table size constant during the test,
// so write latency reflects InnoDB buffer-pool / redo-log throughput, not row
// allocation overhead.
const cpScratch = "_cptest_scratch"

// ─────────────────────────────────────────────────────────────────────────────
// Types
// ─────────────────────────────────────────────────────────────────────────────

// ConnectionPoolTest is the top-level test driver.
// It owns all configuration and delegates execution to runScenario, which is
// called once per worker-level (once in normal mode, multiple times in ramp mode).
type ConnectionPoolTest struct {
	*TestBase
	// Workers is the number of concurrent goroutines that hammer the database.
	// Each goroutine runs independently and reports its own results; the final
	// report aggregates all goroutines into a single latency distribution.
	Workers int
	// MaxOpenConns maps directly to sql.DB.SetMaxOpenConns. When Workers >
	// MaxOpenConns, goroutines must queue inside the Go pool for a free
	// connection slot. That queuing time shows up as elevated Acquire latency
	// and non-zero WaitCount, which is exactly what you want to observe to
	// understand pool pressure.
	MaxOpenConns int
	// MaxIdleConns maps to sql.DB.SetMaxIdleConns. Idle connections are kept
	// open between operations, so the next acquire is essentially free (just
	// a channel read from the pool). Setting this to 0 forces the pool to
	// close connections after every use, making every acquire a full TCP
	// handshake + MySQL auth — useful for stressing the server thread cache.
	MaxIdleConns int
	// ConnMaxLifetimeSec maps to sql.DB.SetConnMaxLifetime. After this many
	// seconds a connection is evicted from the pool and recreated on next use.
	// This forces periodic reconnects even when the pool has idle connections,
	// which increases Threads_created on the server and lowers the thread-cache
	// hit rate. Set to 0 to disable (connections live forever in the pool).
	ConnMaxLifetimeSec int
	// PayloadSize controls the read query: small (~10 B, a single system
	// variable), medium (1 KB REPEAT), large (64 KB), or xlarge (1 MB).
	// Larger payloads shift wall-clock time toward network I/O so you can see
	// how much of Round-trip latency is connection overhead vs data transfer.
	PayloadSize string
	// WriteSize is the number of bytes written per ConnReuse iteration.
	// 0 means read-only. When > 0 a scratch table is created and each worker
	// UPDATEs its own dedicated row, exercising InnoDB redo log and binlog.
	WriteSize int
	// ConnReuse is how many read (and optionally write) SQL statements are
	// executed on a single acquired connection before it is released back to
	// the pool. ConnReuse=1 (the default) gives you the pure acquire+release
	// overhead per query. Higher values amortise the connection cost across
	// multiple queries, as a typical application ORM does, so you can measure
	// "what fraction of my latency is connection overhead vs query execution".
	ConnReuse int
	// RampMode runs a series of scenarios at increasing concurrency levels
	// (1→4→16→...→Workers) instead of a single scenario. This produces a
	// scaling curve: you can see at what worker count throughput saturates and
	// latency P99 starts to blow up.
	RampMode    bool
	WarmupLoops int
	// Duration, when > 0, switches from loop-based execution to time-based:
	// workers run until the deadline expires rather than completing a fixed
	// number of loops. This is useful for steady-state measurements and for
	// matching sysbench's --time flag semantics.
	Duration int
	// ReportInterval controls how often (in seconds) the live interval
	// reporter prints a one-line summary to stdout while the test runs,
	// similar to sysbench's output. Set to 0 to disable interval output;
	// the progress bar is shown instead (in loop-based mode).
	ReportInterval int
}

// cpResult holds all timing measurements for one complete
// acquire → ConnReuse×(read[+write]) → release cycle.
// All durations are in nanoseconds to preserve sub-microsecond precision when
// aggregating across millions of operations.
type cpResult struct {
	acquireNs    int64 // time from db.Conn() call to connection being ready
	readNs       int64 // cumulative time spent inside QueryRowContext across ConnReuse iterations
	writeNs      int64 // cumulative time spent inside ExecContext (writes) across ConnReuse iterations
	releaseNs    int64 // time from conn.Close() call to return completing
	bytesRead    int64 // total bytes returned by read queries (len of scanned string)
	bytesWritten int64 // total bytes sent in write payloads
	err          error // first non-nil error encountered in this cycle; nil means success
}

// cpCounters holds running totals that the worker goroutines update atomically.
// The interval reporter goroutine reads and resets these every ReportInterval
// seconds using atomic.SwapInt64, which simultaneously returns the old value and
// sets the counter to zero — no mutex needed and no samples are lost between the
// swap and the next worker increment.
type cpCounters struct {
	ops          int64 // total completed cycles since last reset
	errors       int64 // cycles that returned a non-nil error
	acquireNsSum int64 // sum of acquireNs across all ops (used to compute mean)
	releaseNsSum int64 // sum of releaseNs across all ops
	readNsSum    int64 // sum of readNs across all ops
	writeNsSum   int64 // sum of writeNs across all ops (0 when WriteSize==0)
	bytesRead    int64 // total bytes received from MySQL
	bytesWritten int64 // total bytes sent to MySQL as write payloads
}

// mysqlThreadStats captures MySQL server-side connection counters. We snapshot
// these before and after each scenario; the deltas tell us exactly how many new
// connections the scenario created and how the server's threading layer handled them.
//
// MySQL supports two threading models, selected by @@thread_handling:
//
//   - "one-thread-per-connection" (default community MySQL): each accepted
//     client connection gets its own dedicated OS thread. The thread cache
//     (thread_cache_size) lets the server reuse OS threads across sequential
//     connections to amortise the cost of OS thread creation.
//
//   - "pool-of-threads" (Percona Server Thread Pool plugin, MySQL Enterprise):
//     a fixed pool of worker threads multiplexes all client connections. No OS
//     thread is ever created per-connection, so Threads_cached is always 0 and
//     thread_cache_size is completely ignored by the server. The meaningful
//     contention signal is the thread pool queue depth, not a cache hit rate.
//
// The collection and reporting code branches on threadHandling so that each
// model shows its own relevant metrics and suppresses the irrelevant ones.
type mysqlThreadStats struct {
	// threadHandling is the value of @@thread_handling at snapshot time.
	// "one-thread-per-connection" → use thread-cache metrics.
	// "pool-of-threads"          → use thread-pool metrics; ignore cache fields.
	threadHandling string

	// Standard status variables — meaningful in both threading models.
	threadsConnected int64 // currently open client connections
	threadsRunning   int64 // connections actively executing SQL (not sleeping)
	connections      int64 // cumulative total connection attempts (monotonically increasing)

	// One-thread-per-connection model only.
	// In pool-of-threads mode these are collected but not displayed, because:
	//   - Threads_cached is always 0 (pool threads are never "cached")
	//   - Threads_created delta is ~0 after pool init (no per-connection threads)
	//   - The cache hit rate formula therefore gives a spurious 100%
	threadsCached   int64 // OS threads sitting in the cache ready for reuse
	threadsCreated  int64 // cumulative OS threads ever spawned (monotonically increasing)
	threadCacheSize int64 // @@thread_cache_size (irrelevant in pool mode)

	// Thread Pool plugin metrics — only populated when threadHandling == "pool-of-threads".
	// These are the meaningful concurrency indicators when the pool plugin is active.
	tpThreads       int64 // Threadpool_threads: total worker threads currently alive in the pool
	tpIdleThreads   int64 // Threadpool_idle_threads: workers waiting for new requests
	tpQueuedQueries int64 // Threadpool_queued_queries: requests waiting because no worker was free
	tpSize          int64 // @@threadpool_size: configured maximum number of pool threads

	// Performance schema socket wait instrumentation.
	// Valid in both threading models: tracks how long the server spent accepting
	// new TCP connections at the socket layer, independent of what thread handles them.
	// Timer values from performance_schema are in picoseconds; we convert to ns on read.
	psConnCount int64 // number of socket accept events recorded
	psConnSumNs int64 // total server-side connection accept time (ns)
	psConnAvgNs int64 // running average reported by performance_schema (ns)
	psConnMaxNs int64 // worst-case single accept observed by the server (ns)
}

// appPoolSnapshot captures sql.DB.Stats() at a point in time. We take snapshots
// before and after each scenario so that the deltas reflect only the scenario's
// activity, not the warmup or previous scenarios.
type appPoolSnapshot struct {
	openConnections int           // total open connections (InUse + Idle)
	inUse           int           // connections currently checked out by goroutines
	idle            int           // connections sitting idle in the pool
	waitCount       int64         // goroutines that had to block waiting for a connection
	waitDuration    time.Duration // total wall time all goroutines spent waiting
	maxIdleClosed   int64         // connections closed because the pool had more than MaxIdleConns idle
	maxLifeClosed   int64         // connections closed because they exceeded ConnMaxLifetime
}

// ─────────────────────────────────────────────────────────────────────────────
// Constructor
// ─────────────────────────────────────────────────────────────────────────────

// NewConnectionPoolTest builds the test from the parsed command-line parameters.
func NewConnectionPoolTest(params config.Params) *ConnectionPoolTest {
	tb := NewTestBase(params)
	return &ConnectionPoolTest{
		TestBase:           tb,
		Workers:            params.Workers,
		MaxOpenConns:       params.MaxOpenConns,
		MaxIdleConns:       params.MaxIdleConns,
		ConnMaxLifetimeSec: params.ConnMaxLifetimeSec,
		PayloadSize:        params.PayloadSize,
		WriteSize:          params.WriteSize,
		ConnReuse:          params.ConnReuse,
		RampMode:           params.RampWorkers,
		WarmupLoops:        params.WarmupLoops,
		Duration:           params.Duration,
		ReportInterval:     params.ReportInterval,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Run — entry point
// ─────────────────────────────────────────────────────────────────────────────

// Run is the main entry point called by main.go after flag parsing.
// It sets up the shared sql.DB connection pool (which is reused across all
// scenarios), optionally creates the write scratch table, runs warmup, then
// iterates over the worker-level sequence. Each iteration calls runScenario,
// which is the unit of measurement and reporting.
func (t *ConnectionPoolTest) Run() {
	t.Init()

	// Parse the host:port URL into components needed by GetConnection.
	host, port, ok := t.SplitIPAndPort(t.Parameters.Url)
	if !ok {
		log.Error("ConnectionPoolTest: invalid URL")
		os.Exit(1)
	}
	portI, err := strconv.Atoi(port)
	if err != nil {
		log.Errorf("ConnectionPoolTest: invalid port: %s", err)
		os.Exit(1)
	}

	cParams := config.ConnectionParameters{
		User:        t.Parameters.User,
		Password:    t.Parameters.Password,
		Host:        host,
		Port:        portI,
		PingTimeout: t.Parameters.PingTimeout,
	}

	ok, node := t.GetConnection(cParams)
	if !ok {
		log.Error("ConnectionPoolTest: cannot connect to MySQL")
		os.Exit(1)
	}
	defer node.Close()

	db := node.Connection

	// Apply pool configuration. These settings are the primary knobs under test:
	// how many connections the pool can have open, how many it keeps idle, and
	// how long before it recycles them. The test deliberately manipulates these
	// relative to the number of worker goroutines so that contention, idle reuse,
	// and forced reconnects can each be observed in isolation.
	db.SetMaxOpenConns(t.MaxOpenConns)
	db.SetMaxIdleConns(t.MaxIdleConns)
	if t.ConnMaxLifetimeSec > 0 {
		db.SetConnMaxLifetime(time.Duration(t.ConnMaxLifetimeSec) * time.Second)
	}
	// Guard against a misconfigured ConnReuse of 0; at least one query must run.
	if t.ConnReuse < 1 {
		t.ConnReuse = 1
	}

	// Build the fully-qualified scratch table name. We qualify it with the schema
	// if one is given so that the table lands in the right database regardless of
	// the connection's default schema. We skip the "mysql" default because writing
	// test data there is undesirable and usually forbidden.
	scratchTable := cpScratch
	if t.Parameters.Schema != "" && t.Parameters.Schema != "mysql" {
		scratchTable = "`" + t.Parameters.Schema + "`." + cpScratch
	}

	// Create the scratch table once up-front. Seeding (TRUNCATE + INSERT) happens
	// per-scenario because each concurrency level requires a different number of
	// pre-seeded rows (one per worker).
	if t.WriteSize > 0 {
		if err := t.setupScratchTable(db, scratchTable); err != nil {
			log.Errorf("ConnectionPoolTest: cannot create scratch table %s: %v", scratchTable, err)
			os.Exit(1)
		}
	}

	t.printHeader(db)

	// Warmup runs a small number of real connections before measurement begins.
	// This pre-heats the Go pool (so idle connections are already open), primes
	// the MySQL thread cache, and lets TCP slow-start settle — all factors that
	// would otherwise inflate the first few measurements.
	if t.WarmupLoops > 0 {
		t.warmup(db)
	}

	// Print the CSV column-name comment lines before any data rows are emitted.
	if t.ReportCSV {
		t.printCSVHeaders()
	}

	// Run one scenario per worker level. In ramp mode this iterates 1→4→16→…→N;
	// in normal mode it runs a single scenario at t.Workers.
	for _, w := range t.workerLevels() {
		// Re-seed the scratch table before each scenario so every worker has
		// exactly one dedicated row to UPDATE (no shared row → no lock contention
		// between workers during the write phase).
		if t.WriteSize > 0 {
			if err := t.seedScratchTable(db, scratchTable, w); err != nil {
				log.Errorf("ConnectionPoolTest: cannot seed scratch table: %v", err)
				os.Exit(1)
			}
		}
		t.runScenario(db, w, scratchTable)
	}

	os.Exit(0)
}

// workerLevels returns the sequence of concurrency levels to test.
// In ramp mode the predefined ladder is trimmed to levels ≤ Workers and the
// exact Workers value is always appended as the final entry so the user's
// requested maximum is always included. In non-ramp mode a single-element
// slice is returned.
func (t *ConnectionPoolTest) workerLevels() []int {
	if !t.RampMode {
		return []int{t.Workers}
	}
	levels := []int{1, 4, 16, 32, 64, 128, 256, 512, 1024, 2048, 3096}
	var out []int
	for _, l := range levels {
		if l <= t.Workers {
			out = append(out, l)
		}
	}
	if len(out) == 0 || out[len(out)-1] != t.Workers {
		out = append(out, t.Workers)
	}
	return out
}

// ─────────────────────────────────────────────────────────────────────────────
// Scratch table — write test support
// ─────────────────────────────────────────────────────────────────────────────

// setupScratchTable creates the scratch table if it does not already exist.
// The table has one row per worker keyed by worker_id. Using a PRIMARY KEY on
// worker_id means each UPDATE hits exactly one row via the clustered index with
// no secondary index maintenance — the write cost reflects pure InnoDB redo-log
// and buffer-pool pressure, not index structure overhead.
func (t *ConnectionPoolTest) setupScratchTable(db *sql.DB, table string) error {
	_, err := db.Exec(fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
			worker_id INT NOT NULL,
			data      LONGBLOB NOT NULL,
			PRIMARY KEY (worker_id)
		) ENGINE=InnoDB`, table))
	return err
}

// seedScratchTable truncates the table and inserts one row per worker, each
// pre-populated with a payload of the target write size. Seeding happens
// outside the measurement window so INSERT overhead is not included in the
// reported write latency. During the scenario each worker issues only UPDATE
// statements against its own row.
func (t *ConnectionPoolTest) seedScratchTable(db *sql.DB, table string, workers int) error {
	// TRUNCATE is a DDL operation under the hood (InnoDB) — it drops and
	// recreates the table segment, so it is much faster than DELETE for large
	// tables and does not write individual row undo records.
	if _, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %s", table)); err != nil {
		return err
	}
	payload := strings.Repeat("w", t.WriteSize)
	for i := 0; i < workers; i++ {
		if _, err := db.Exec(
			fmt.Sprintf("INSERT INTO %s (worker_id, data) VALUES (?, ?)", table),
			i, payload,
		); err != nil {
			return err
		}
	}
	return nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Warmup
// ─────────────────────────────────────────────────────────────────────────────

// warmup runs a sequential series of acquire-query-release cycles to stabilise
// the environment before measurement starts. Without warmup the first few
// operations show inflated acquire times because:
//   - The Go pool has no idle connections yet; every acquire spawns a new TCP
//     connection and MySQL thread.
//   - The MySQL thread cache is empty after a fresh server start.
//   - TCP slow-start means the first packets to a cold server are paced slowly.
//
// Sequential (single-goroutine) warmup is intentional: we want the pool to
// settle to MaxIdleConns idle connections without racing goroutines competing
// for slots, which would produce noisy warm-up measurements.
func (t *ConnectionPoolTest) warmup(db *sql.DB) {
	if !t.ReportCSV {
		fmt.Printf("Warming up pool (%d iterations)...\n", t.WarmupLoops)
	}
	ctx := context.Background()
	q := t.readQuery()
	for i := 0; i < t.WarmupLoops; i++ {
		conn, err := db.Conn(ctx)
		if err != nil {
			continue
		}
		var data string
		conn.QueryRowContext(ctx, q).Scan(&data) //nolint
		conn.Close()
	}
	if !t.ReportCSV {
		fmt.Println("Warmup complete.")
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Core scenario execution
// ─────────────────────────────────────────────────────────────────────────────

// runScenario is the heart of the test. It orchestrates:
//  1. A pre-scenario MySQL snapshot (before counters)
//  2. A result-drainer goroutine (keeps workers unblocked)
//  3. An optional interval-reporter goroutine (sysbench-style live output)
//  4. The worker goroutines (parallel load generators)
//  5. A post-scenario MySQL snapshot (after counters)
//  6. Report generation and histogram printing
func (t *ConnectionPoolTest) runScenario(db *sql.DB, workers int, scratchTable string) {
	timeBased := t.Duration > 0

	// useBar decides whether to render an in-place progress bar. The bar and the
	// interval reporter would overwrite each other if both were active, so the bar
	// is suppressed whenever the interval reporter is running or the run is
	// time-based (where the total operation count is unknown up-front).
	useBar := !t.ReportCSV && !timeBased && t.ReportInterval == 0

	// Build the context that controls worker lifetime.
	// In time-based mode a WithTimeout context causes all goroutines to stop
	// cleanly when the deadline fires — no explicit done-channel needed.
	// In loop-based mode a plain cancelable context is used; goroutines stop
	// naturally when their loop counter reaches t.Loops.
	var ctx context.Context
	var cancel context.CancelFunc
	if timeBased {
		ctx, cancel = context.WithTimeout(context.Background(),
			time.Duration(t.Duration)*time.Second)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	totalOps := int64(workers * t.Loops) // only meaningful in loop-based mode

	readQuery := t.readQuery()
	writePayload := strings.Repeat("w", t.WriteSize)

	// Snapshot MySQL and app-pool state before the scenario starts. All metrics
	// in the report are computed as (after − before) so that any pre-existing
	// activity on the server does not inflate our numbers.
	mysqlBefore := t.captureMySQLStats(db)
	appBefore := snapshotAppPool(db)

	// The result channel uses a small fixed buffer. A dedicated drainer goroutine
	// continuously reads from it and appends to a slice. This decouples workers
	// from the result collection path: workers never block on a full channel even
	// if the drainer is momentarily slow, and we avoid allocating a huge
	// pre-sized channel for time-based runs where the total operation count is
	// unknown.
	resultsCh := make(chan cpResult, 512)
	var (
		allResults []cpResult
		drainMu    sync.Mutex
		drainerWg  sync.WaitGroup
	)
	drainerWg.Add(1)
	go func() {
		defer drainerWg.Done()
		for r := range resultsCh {
			drainMu.Lock()
			allResults = append(allResults, r)
			drainMu.Unlock()
		}
	}()

	// cpCounters is updated atomically by every worker goroutine and read
	// (and reset) by the interval reporter on each tick.
	var counters cpCounters
	// opCounter tracks the total completed operations for the progress bar.
	// It is separate from counters.ops to avoid the bar logic interfering with
	// the interval reporter's atomic-swap reset pattern.
	var opCounter int64

	// Update the progress bar at most ~20 times across the full run.
	// The floor of 50 prevents spamming on very short runs;
	// the ceiling of 5000 prevents silently doing nothing on very long ones.
	progressEvery := totalOps / 20
	if progressEvery < 50 {
		progressEvery = 50
	}
	if progressEvery > 5000 {
		progressEvery = 5000
	}

	// Start the interval reporter if requested. It runs in a separate goroutine
	// and is stopped by closing the stopReporter channel after workers finish.
	stopReporter := make(chan struct{})
	var reporterWg sync.WaitGroup
	if t.ReportInterval > 0 {
		t.startIntervalReporter(stopReporter, &reporterWg, &counters, workers)
	}

	startTime := time.Now()
	var wg sync.WaitGroup

	// Launch one goroutine per worker. Each goroutine runs independently;
	// synchronisation between workers happens only through the shared sql.DB
	// pool (which is intentional — pool contention is one of the things we
	// are measuring).
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			if timeBased {
				// Time-based mode: keep executing until the context deadline fires.
				// Checking ctx.Done() before each operation (rather than after)
				// means we do not start a new cycle after the deadline, which keeps
				// the elapsed time accurate.
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}
					r := t.executeOp(ctx, db, readQuery, writePayload, scratchTable, workerID)
					resultsCh <- r
					t.updateCounters(&counters, r)
					if t.Sleep > 0 {
						time.Sleep(time.Duration(t.Sleep) * time.Millisecond)
					}
				}
			} else {
				// Loop-based mode: execute exactly t.Loops full cycles.
				for i := 0; i < t.Loops; i++ {
					r := t.executeOp(ctx, db, readQuery, writePayload, scratchTable, workerID)
					resultsCh <- r
					t.updateCounters(&counters, r)
					// Increment the shared counter and update the progress bar.
					// atomic.AddInt64 is safe across goroutines without a mutex.
					cur := atomic.AddInt64(&opCounter, 1)
					if useBar && cur%progressEvery == 0 {
						cpProgressBar(cur, totalOps, time.Since(startTime))
					}
					if t.Sleep > 0 {
						time.Sleep(time.Duration(t.Sleep) * time.Millisecond)
					}
				}
			}
		}(w)
	}

	// Block until every worker has finished. After this point no more results
	// will be sent to resultsCh or written to counters.
	wg.Wait()
	elapsed := time.Since(startTime)

	// Print the completed progress bar and advance to a new line so the
	// scenario report that follows prints cleanly below it.
	if useBar {
		cpProgressBar(totalOps, totalOps, elapsed)
		fmt.Println()
	}

	// Signal the interval reporter to stop and wait for its final tick to flush.
	// We close stopReporter before closing resultsCh so the reporter has a
	// chance to print the last partial interval before the report takes over.
	close(stopReporter)
	reporterWg.Wait()

	// Close the results channel now that no more workers are running, which
	// causes the drainer goroutine to exit its range loop. Wait for it to
	// finish so allResults is fully populated before we call printScenarioReport.
	close(resultsCh)
	drainerWg.Wait()

	// Snapshot MySQL and app-pool state after all workers have finished.
	mysqlAfter := t.captureMySQLStats(db)
	appAfter := snapshotAppPool(db)

	t.printScenarioReport(workers, allResults, elapsed, mysqlBefore, mysqlAfter, appBefore, appAfter)
	// Histograms are only meaningful in human-readable mode; CSV consumers
	// should derive distributions from the raw per-operation data if needed.
	if !t.ReportCSV {
		t.printHistograms(allResults)
	}
}

// executeOp performs one complete connection lifecycle:
//
//	db.Conn()  →  ConnReuse × (SELECT [+ UPDATE])  →  conn.Close()
//
// Each phase is timed independently so the report can show what fraction of
// round-trip latency is pool acquisition, query execution, and release.
//
// Design note: we call db.Conn() explicitly rather than db.QueryRow() because
// QueryRow leases a connection internally without exposing the acquisition
// timestamp. We need the before/after timestamps to separate acquire time from
// query time.
func (t *ConnectionPoolTest) executeOp(
	ctx context.Context,
	db *sql.DB,
	readQuery, writePayload, scratchTable string,
	workerID int,
) cpResult {
	r := cpResult{}

	// --- Acquire phase ---
	// db.Conn() checks out a connection from the pool.
	// If MaxOpenConns has been reached, db.Conn() blocks here until another
	// goroutine releases a connection. That blocking time is captured in
	// acquireNs and surfaced as pool contention in the report.
	t1 := time.Now()
	conn, err := db.Conn(ctx)
	r.acquireNs = time.Since(t1).Nanoseconds()
	if err != nil {
		// Context cancellation (time-based deadline) or a genuine connection
		// error. Either way, record the error and return without attempting
		// any queries; the error will be counted separately from latency.
		r.err = err
		return r
	}

	// --- Query phase (repeated ConnReuse times) ---
	// Running multiple queries on the same connection before releasing it
	// models application behaviour where an ORM or query layer executes
	// several statements inside a single connection checkout (e.g. a request
	// handler that runs multiple SELECTs before responding). By varying
	// ConnReuse you can measure the marginal cost of additional queries vs
	// the fixed cost of acquiring and releasing the connection.
	for i := 0; i < t.ConnReuse; i++ {
		// Read sub-phase.
		tq := time.Now()
		var data string
		if qErr := conn.QueryRowContext(ctx, readQuery).Scan(&data); qErr != nil && r.err == nil {
			r.err = qErr
		}
		r.readNs += time.Since(tq).Nanoseconds()
		// len(data) gives us the byte count of the returned string, which is
		// the actual data transferred over the wire (excluding protocol overhead).
		r.bytesRead += int64(len(data))

		// Write sub-phase — only when --writeSize > 0.
		// Each worker updates its own dedicated row (worker_id = workerID), so
		// there is no row-level lock contention between workers. The cost we
		// measure is purely InnoDB redo-log writes, buffer-pool dirty-page
		// management, and optionally binlog I/O.
		if t.WriteSize > 0 {
			wq := fmt.Sprintf("UPDATE %s SET data=? WHERE worker_id=?", scratchTable)
			tw := time.Now()
			if _, wErr := conn.ExecContext(ctx, wq, writePayload, workerID); wErr != nil && r.err == nil {
				r.err = wErr
			}
			r.writeNs += time.Since(tw).Nanoseconds()
			r.bytesWritten += int64(t.WriteSize)
		}
	}

	// --- Release phase ---
	// conn.Close() returns the connection to the pool (it does not close the
	// underlying TCP connection unless the pool is already at MaxIdleConns).
	// Release time is normally very small (microseconds) because it is just a
	// channel send inside the pool. A spike here would indicate Go runtime
	// scheduling pressure or an unexpectedly slow pool implementation.
	t3 := time.Now()
	conn.Close()
	r.releaseNs = time.Since(t3).Nanoseconds()

	return r
}

// updateCounters adds a completed operation's measurements to the shared
// atomic counters. All fields use atomic.AddInt64 to avoid mutex overhead on
// the hot path. The interval reporter drains these counters with atomic.SwapInt64
// on each tick, which atomically returns the accumulated value and resets the
// counter to zero — ensuring no overlap or double-counting between intervals.
func (t *ConnectionPoolTest) updateCounters(c *cpCounters, r cpResult) {
	atomic.AddInt64(&c.ops, 1)
	if r.err != nil {
		atomic.AddInt64(&c.errors, 1)
	}
	atomic.AddInt64(&c.acquireNsSum, r.acquireNs)
	atomic.AddInt64(&c.releaseNsSum, r.releaseNs)
	atomic.AddInt64(&c.readNsSum, r.readNs)
	atomic.AddInt64(&c.writeNsSum, r.writeNs)
	atomic.AddInt64(&c.bytesRead, r.bytesRead)
	atomic.AddInt64(&c.bytesWritten, r.bytesWritten)
}

// ─────────────────────────────────────────────────────────────────────────────
// Interval reporter (sysbench-style)
// ─────────────────────────────────────────────────────────────────────────────

// startIntervalReporter launches a background goroutine that prints one summary
// line every ReportInterval seconds while the scenario is running. This mirrors
// sysbench's --report-interval output and lets you observe how throughput and
// latency evolve over time rather than seeing only the final aggregate.
//
// The reporter uses atomic.SwapInt64 to simultaneously read and reset each
// counter. This means each interval's averages are computed from that interval's
// operations only — not a cumulative average — making it easy to spot transient
// spikes or ramp-up effects.
func (t *ConnectionPoolTest) startIntervalReporter(
	stop chan struct{},
	wg *sync.WaitGroup,
	c *cpCounters,
	workers int,
) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(time.Duration(t.ReportInterval) * time.Second)
		defer ticker.Stop()
		elapsed := 0

		// Print the column header once before the first data line.
		if !t.ReportCSV {
			fmt.Printf("\n  %-6s %-8s %-7s %-10s %-11s %-11s %-11s %-11s %-10s %-10s\n",
				"t(s)", "ops", "errors", "tps",
				"acq_avg_µs", "rel_avg_µs", "rd_avg_µs", "wr_avg_µs",
				"rd_KB/s", "wr_KB/s")
		}

		for {
			select {
			case <-ticker.C:
				elapsed += t.ReportInterval
				// dt is the real wall-clock duration of this interval, used as
				// the denominator for TPS and throughput calculations.
				dt := float64(t.ReportInterval)

				// SwapInt64 atomically returns the current value and sets it to 0.
				// Using Swap instead of Load+Store means we never lose counts that
				// a worker adds between the Load and the Store.
				ops := atomic.SwapInt64(&c.ops, 0)
				errs := atomic.SwapInt64(&c.errors, 0)
				acqSum := atomic.SwapInt64(&c.acquireNsSum, 0)
				relSum := atomic.SwapInt64(&c.releaseNsSum, 0)
				rdSum := atomic.SwapInt64(&c.readNsSum, 0)
				wrSum := atomic.SwapInt64(&c.writeNsSum, 0)
				bytesR := atomic.SwapInt64(&c.bytesRead, 0)
				bytesW := atomic.SwapInt64(&c.bytesWritten, 0)

				// Convert summed nanoseconds to average microseconds.
				// Guard against ops==0 (no completions in this interval) to
				// avoid a divide-by-zero when the pool is fully saturated and
				// no operation managed to finish within the reporting window.
				var acqAvg, relAvg, rdAvg, wrAvg float64
				if ops > 0 {
					acqAvg = float64(acqSum) / float64(ops) / 1000
					relAvg = float64(relSum) / float64(ops) / 1000
					rdAvg = float64(rdSum) / float64(ops) / 1000
					wrAvg = float64(wrSum) / float64(ops) / 1000
				}
				tps := float64(ops) / dt
				rdKBs := float64(bytesR) / dt / 1024
				wrKBs := float64(bytesW) / dt / 1024

				// In CSV mode prefix with "interval," so that a grep or awk
				// script can split interval lines from summary lines.
				if t.ReportCSV {
					fmt.Printf("interval,%d,%d,%d,%.2f,%.1f,%.1f,%.1f,%.1f,%.2f,%.2f\n",
						elapsed, ops, errs, tps,
						acqAvg, relAvg, rdAvg, wrAvg,
						rdKBs, wrKBs)
				} else {
					fmt.Printf("  %-6d %-8d %-7d %-10.1f %-11.1f %-11.1f %-11.1f %-11.1f %-10.2f %-10.2f\n",
						elapsed, ops, errs, tps,
						acqAvg, relAvg, rdAvg, wrAvg,
						rdKBs, wrKBs)
				}

			case <-stop:
				// The main goroutine closed stopReporter after all workers finished.
				// Exit cleanly so reporterWg.Wait() can return.
				return
			}
		}
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
// MySQL stats capture
// ─────────────────────────────────────────────────────────────────────────────

// captureMySQLStats reads server-side connection counters and detects the
// active threading model (one-thread-per-connection vs Thread Pool plugin).
// The correct set of metrics is collected for each model so the report never
// displays misleading values (e.g. a thread-cache hit rate of "100%" when
// the thread cache is completely bypassed by the pool plugin).
//
// We try performance_schema.global_status first (structured, server-side
// filtering), falling back to SHOW GLOBAL STATUS for restricted environments
// or servers older than 5.7.
func (t *ConnectionPoolTest) captureMySQLStats(db *sql.DB) mysqlThreadStats {
	stats := mysqlThreadStats{}
	ctx := context.Background()

	// ── Step 1: detect threading model ───────────────────────────────────────
	// @@thread_handling is set by the server at startup based on which plugin
	// is active. The two values we handle are:
	//   "one-thread-per-connection" — default community MySQL
	//   "pool-of-threads"           — Percona Server Thread Pool plugin,
	//                                  MySQL Enterprise Thread Pool
	// Any read error (very old server, restricted user) leaves threadHandling
	// empty, which we treat as one-thread-per-connection for backward compat.
	db.QueryRowContext(ctx, "SELECT @@thread_handling").Scan(&stats.threadHandling) //nolint
	usingThreadPool := stats.threadHandling == "pool-of-threads"

	// ── Step 2: collect common status variables ───────────────────────────────
	// These variables are meaningful in both threading models.
	rows, err := db.QueryContext(ctx,
		"SELECT variable_name, variable_value "+
			"FROM performance_schema.global_status "+
			"WHERE variable_name IN "+
			"('Threads_connected','Threads_running','Threads_cached','Threads_created','Connections',"+
			"'Threadpool_threads','Threadpool_idle_threads','Threadpool_queued_queries')")
	if err != nil {
		// Fallback: some configurations restrict performance_schema SELECT.
		rows, err = db.QueryContext(ctx,
			"SHOW GLOBAL STATUS WHERE Variable_name IN "+
				"('Threads_connected','Threads_running','Threads_cached','Threads_created','Connections',"+
				"'Threadpool_threads','Threadpool_idle_threads','Threadpool_queued_queries')")
		if err != nil {
			return stats
		}
	}
	defer rows.Close()

	for rows.Next() {
		var name string
		var val int64
		if err := rows.Scan(&name, &val); err != nil {
			continue
		}
		switch name {
		case "Threads_connected":
			stats.threadsConnected = val
		case "Threads_running":
			stats.threadsRunning = val
		case "Connections":
			stats.connections = val

		// One-thread-per-connection fields.
		// In pool-of-threads mode: Threads_cached is always 0 (the pool does not
		// cache threads between connections — it keeps them alive in the pool
		// permanently). Threads_created delta is ~0 after server start because
		// pool worker threads are created at init time, not per-connection.
		// We collect these anyway but suppress them from the report in pool mode
		// to avoid the misleading "Threads_cached=0" output.
		case "Threads_cached":
			stats.threadsCached = val
		case "Threads_created":
			stats.threadsCreated = val

		// Thread Pool plugin fields (Percona Server / MySQL Enterprise).
		// These are NULL / missing in one-thread-per-connection mode, which is
		// fine — the zero values are never displayed in that mode.
		case "Threadpool_threads":
			stats.tpThreads = val
		case "Threadpool_idle_threads":
			stats.tpIdleThreads = val
		case "Threadpool_queued_queries":
			// This is the key contention indicator in thread pool mode.
			// A non-zero queue means incoming requests had to wait because all
			// worker threads were busy — the thread pool equivalent of
			// sql.DB WaitCount on the application side.
			stats.tpQueuedQueries = val
		}
	}

	// ── Step 3: collect system variables that need separate queries ───────────
	if usingThreadPool {
		// @@threadpool_size is the configured number of worker threads.
		// Percona Server uses 'threadpool_size'; MySQL Enterprise uses the same.
		// If the variable is absent (e.g. plugin not fully initialised), the
		// zero value is safe — we will just omit the "size" from the display.
		db.QueryRowContext(ctx, "SELECT @@threadpool_size").Scan(&stats.tpSize) //nolint
	} else {
		// @@thread_cache_size is a system variable, not a status variable,
		// so it does not appear in SHOW GLOBAL STATUS.
		db.QueryRowContext(ctx, "SELECT @@thread_cache_size").Scan(&stats.threadCacheSize) //nolint
	}

	// ── Step 4: performance_schema socket accept latency ─────────────────────
	// The 'wait/io/socket/sql/client_connection' event measures how long the
	// server spent accepting new TCP connections at the socket layer.  This is
	// independent of the threading model and valid in both modes.
	//
	// Performance_schema timer values are in picoseconds (10^-12 s).
	// We divide by 1000 to convert to nanoseconds before storing.
	var sumPs, avgPs, maxPs int64
	if err := db.QueryRowContext(ctx,
		"SELECT COUNT_STAR, SUM_TIMER_WAIT, AVG_TIMER_WAIT, MAX_TIMER_WAIT "+
			"FROM performance_schema.events_waits_summary_global_by_event_name "+
			"WHERE EVENT_NAME = 'wait/io/socket/sql/client_connection'",
	).Scan(&stats.psConnCount, &sumPs, &avgPs, &maxPs); err == nil {
		stats.psConnSumNs = sumPs / 1000
		stats.psConnAvgNs = avgPs / 1000
		stats.psConnMaxNs = maxPs / 1000
	}

	return stats
}

// snapshotAppPool wraps db.Stats() into our own struct so that the caller does
// not need to import database/sql for field access, and so that before/after
// snapshots can be diffed with simple arithmetic.
func snapshotAppPool(db *sql.DB) appPoolSnapshot {
	s := db.Stats()
	return appPoolSnapshot{
		openConnections: s.OpenConnections,
		inUse:           s.InUse,
		idle:            s.Idle,
		waitCount:       s.WaitCount,
		waitDuration:    s.WaitDuration,
		maxIdleClosed:   s.MaxIdleClosed,
		maxLifeClosed:   s.MaxLifetimeClosed,
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Report output
// ─────────────────────────────────────────────────────────────────────────────

// printScenarioReport computes all derived metrics from the raw cpResult slice
// and prints the scenario summary in either human-readable or CSV format.
//
// The raw latency slices are sorted in-place so that percentile calculations
// are O(1) index lookups rather than a second O(n log n) sort. Sorting destroys
// the original order but that is fine — we only need the per-sample values for
// statistical aggregation, not for any ordered processing.
func (t *ConnectionPoolTest) printScenarioReport(
	workers int,
	results []cpResult,
	elapsed time.Duration,
	mysqlBefore, mysqlAfter mysqlThreadStats,
	appBefore, appAfter appPoolSnapshot,
) {
	// Separate successful operations from errors. Only successful results
	// contribute to latency statistics; errors are counted and reported
	// separately so that a high error rate does not artificially deflate
	// the latency percentiles.
	var valid []cpResult
	errCount := 0
	for _, r := range results {
		if r.err == nil {
			valid = append(valid, r)
		} else {
			errCount++
		}
	}

	n := len(valid)
	if n == 0 {
		fmt.Printf("workers=%d: all %d operations failed\n", workers, errCount)
		return
	}

	// Build per-phase latency slices. Keeping them separate allows independent
	// sorting and percentile computation for each phase, which is important
	// because the distributions differ significantly — acquire can be bimodal
	// (fast pool hit vs slow pool wait), while release is typically unimodal
	// and very tight.
	acqNs := make([]int64, n)
	relNs := make([]int64, n)
	rdNs := make([]int64, n)
	wrNs := make([]int64, n)
	totNs := make([]int64, n)
	var totalBytesR, totalBytesW int64

	for i, r := range valid {
		acqNs[i] = r.acquireNs
		relNs[i] = r.releaseNs
		rdNs[i] = r.readNs
		wrNs[i] = r.writeNs
		// Round-trip is the end-to-end wall time visible to the caller:
		// acquire + all queries + release.
		totNs[i] = r.acquireNs + r.readNs + r.writeNs + r.releaseNs
		totalBytesR += r.bytesRead
		totalBytesW += r.bytesWritten
	}

	// Sort each slice ascending so cpPct() can use direct index arithmetic.
	sort.Slice(acqNs, func(i, j int) bool { return acqNs[i] < acqNs[j] })
	sort.Slice(relNs, func(i, j int) bool { return relNs[i] < relNs[j] })
	sort.Slice(rdNs, func(i, j int) bool { return rdNs[i] < rdNs[j] })
	sort.Slice(wrNs, func(i, j int) bool { return wrNs[i] < wrNs[j] })
	sort.Slice(totNs, func(i, j int) bool { return totNs[i] < totNs[j] })

	sec := elapsed.Seconds()
	// TPS = cycles (acquire+release pairs) per second.
	tps := float64(n) / sec
	// QPS = individual SQL statements per second. Each cycle runs ConnReuse reads
	// and, if writes are enabled, ConnReuse writes. Multiply accordingly.
	queriesPerOp := float64(t.ConnReuse)
	if t.WriteSize > 0 {
		queriesPerOp *= 2 // one read + one write per ConnReuse iteration
	}
	qps := tps * queriesPerOp
	rdKBs := float64(totalBytesR) / sec / 1024
	wrKBs := float64(totalBytesW) / sec / 1024

	// Determine the threading model from the before-snapshot.
	// We use the before value because it reflects the server state at the
	// start of the scenario; the after value should be identical (thread_handling
	// cannot change at runtime), but using before is safer.
	usingThreadPool := mysqlBefore.threadHandling == "pool-of-threads"

	// Thread-cache hit rate — only meaningful in one-thread-per-connection mode.
	//
	// Formula: (ΔConnections − ΔThreads_created) / ΔConnections × 100
	//   100% = every new connection reused a cached OS thread (ideal)
	//   <80% = thread_cache_size is too small for the concurrency level
	//
	// In pool-of-threads mode this formula always returns ~100% because
	// ΔThreads_created ≈ 0 (pool workers are created at startup, not per-connection).
	// That "100%" is NOT a cache hit — it is a meaningless artefact of the model.
	// We skip the calculation entirely in pool mode to avoid displaying it.
	deltaConns := mysqlAfter.connections - mysqlBefore.connections
	deltaCreated := mysqlAfter.threadsCreated - mysqlBefore.threadsCreated
	var cacheHitPct float64
	if !usingThreadPool && deltaConns > 0 {
		cacheHitPct = float64(deltaConns-deltaCreated) / float64(deltaConns) * 100
		if cacheHitPct < 0 {
			cacheHitPct = 0 // clamp: can happen on integer wrap-around in very long runs
		}
	}

	// App pool wait deltas: non-zero WaitCount means goroutines queued inside
	// the Go pool waiting for a free connection slot, which is direct evidence
	// of pool exhaustion. WaitDuration is the total wall time wasted waiting.
	deltaWaitCount := appAfter.waitCount - appBefore.waitCount
	deltaWaitDur := appAfter.waitDuration - appBefore.waitDuration

	// Performance schema socket delta: how many new connections MySQL accepted
	// during the scenario and how long that took on the server side.
	psDeltaCount := mysqlAfter.psConnCount - mysqlBefore.psConnCount
	psDeltaSumNs := mysqlAfter.psConnSumNs - mysqlBefore.psConnSumNs
	var psAvgConnNs int64
	if psDeltaCount > 0 {
		psAvgConnNs = psDeltaSumNs / psDeltaCount
	}

	if t.ReportCSV {
		// CSV output: "summary," prefix distinguishes these rows from "interval,"
		// rows so both can coexist in a single redirect file and be split by grep.
		//
		// cache_hit_pct is always 0.0 in pool-of-threads mode; consumers should
		// check thread_handling to decide whether to interpret that column.
		// tp_queued_delta and tp_threads are 0 in one-thread-per-connection mode.
		deltaQueued := mysqlAfter.tpQueuedQueries - mysqlBefore.tpQueuedQueries
		fmt.Printf("summary,%d,%d,%d,%s,%d,%d,%.3f,"+
			"%d,%d,"+
			"%d,%d,%.2f,%.2f,%.2f,%.2f,"+
			"%d,%d,%d,%d,%d,"+
			"%d,%d,%d,%d,%d,"+
			"%d,%d,%d,%d,%d,"+
			"%d,%d,%d,%d,%d,"+
			"%d,%d,%d,%d,%.1f,"+
			"%s,%d,%d,%d,"+
			"%d,%d,%d,%d,%.1f,"+
			"%d,%d,%d\n",
			workers, t.Loops, t.Duration, t.PayloadSize, t.WriteSize, t.ConnReuse, sec,
			t.MaxOpenConns, t.MaxIdleConns,
			n, errCount, tps, qps, rdKBs, wrKBs,
			// acquire ns percentiles
			int64(cpMean(acqNs)), cpPct(acqNs, 50), cpPct(acqNs, 95), cpPct(acqNs, 99), acqNs[n-1],
			// release ns percentiles
			int64(cpMean(relNs)), cpPct(relNs, 50), cpPct(relNs, 95), cpPct(relNs, 99), relNs[n-1],
			// read ns percentiles
			int64(cpMean(rdNs)), cpPct(rdNs, 50), cpPct(rdNs, 95), cpPct(rdNs, 99), rdNs[n-1],
			// write ns percentiles (all zeros when WriteSize == 0)
			int64(cpMean(wrNs)), cpPct(wrNs, 50), cpPct(wrNs, 95), cpPct(wrNs, 99), wrNs[n-1],
			// app pool snapshot
			appAfter.openConnections, appAfter.inUse, appAfter.idle,
			deltaWaitCount, float64(deltaWaitDur.Milliseconds()),
			// threading model and server-side concurrency metrics
			mysqlBefore.threadHandling,
			mysqlBefore.threadsConnected, mysqlBefore.threadsCreated, mysqlAfter.threadsConnected,
			// thread cache (one-thread-per-connection) or thread pool, mutually exclusive
			mysqlBefore.threadsCached, mysqlAfter.threadsCached, deltaCreated, mysqlBefore.threadCacheSize, cacheHitPct,
			// thread pool queue (zero in one-thread-per-connection mode)
			mysqlBefore.tpQueuedQueries, mysqlAfter.tpQueuedQueries, deltaQueued,
		)
		return
	}

	// ── Human-readable report ─────────────────────────────────────────────────

	const bar = "══════════════════════════════════════════════════════════════════════"
	const sep = "──────────────────────────────────────────────────────────────────────"

	fmt.Println()
	fmt.Println(bar)
	fmt.Printf("  Workers: %-5d | Ops: %-8d | Duration: %s\n",
		workers, n, elapsed.Round(time.Millisecond))
	fmt.Printf("  Read: %-8s | Write: %-7d bytes | ConnReuse: %d queries/conn\n",
		t.PayloadSize, t.WriteSize, t.ConnReuse)
	fmt.Printf("  App Pool: MaxOpen=%-4d MaxIdle=%-4d | Errors: %d\n",
		t.MaxOpenConns, t.MaxIdleConns, errCount)
	fmt.Printf("  Throughput: %.1f TPS | %.1f QPS | %.2f KB/s read | %.2f KB/s write\n",
		tps, qps, rdKBs, wrKBs)
	fmt.Println(sep)

	// Latency table. All values displayed in microseconds for readability;
	// the raw nanosecond values are available in CSV mode for precision.
	fmt.Printf("  %-12s %12s %12s %12s %12s %12s\n",
		"Phase", "Avg (µs)", "P50 (µs)", "P95 (µs)", "P99 (µs)", "Max (µs)")

	printPhaseRow := func(label string, ns []int64) {
		if len(ns) == 0 {
			return
		}
		fmt.Printf("  %-12s %12d %12d %12d %12d %12d\n",
			label,
			int64(cpMean(ns))/1000,
			cpPct(ns, 50)/1000,
			cpPct(ns, 95)/1000,
			cpPct(ns, 99)/1000,
			ns[len(ns)-1]/1000)
	}
	printPhaseRow("Acquire", acqNs)
	printPhaseRow("Read", rdNs)
	if t.WriteSize > 0 {
		printPhaseRow("Write", wrNs)
	}
	printPhaseRow("Release", relNs)
	printPhaseRow("Round-trip", totNs)

	// ── App pool section ──────────────────────────────────────────────────────
	fmt.Println(sep)
	fmt.Println("  App Pool (sql.DB):")
	fmt.Printf("    OpenConns=%-4d  InUse=%-4d  Idle=%-4d\n",
		appAfter.openConnections, appAfter.inUse, appAfter.idle)
	// WaitCount Δ: how many times a goroutine had to block in db.Conn().
	// WaitDuration Δ: total time those goroutines spent waiting.
	// If WaitCount > 0, the pool was exhausted at some point during the scenario.
	fmt.Printf("    WaitCount(Δ)=%-8d  WaitDuration(Δ)=%s\n",
		deltaWaitCount, deltaWaitDur.Round(time.Millisecond))
	// MaxIdleClosed Δ: connections discarded because the idle count would have
	// exceeded MaxIdleConns. High values mean you set MaxIdleConns too low for
	// the current traffic and are paying connection creation costs unnecessarily.
	// MaxLifetimeClosed Δ: connections evicted because of ConnMaxLifetime.
	// High values combined with a low cache hit rate confirm that lifetime-based
	// recycling is forcing MySQL to spawn new OS threads.
	fmt.Printf("    MaxIdleClosed(Δ)=%-4d  MaxLifetimeClosed(Δ)=%d\n",
		appAfter.maxIdleClosed-appBefore.maxIdleClosed,
		appAfter.maxLifeClosed-appBefore.maxLifeClosed)

	// ── MySQL server section — branches on threading model ───────────────────
	fmt.Println(sep)

	if usingThreadPool {
		// Thread Pool plugin mode.
		// The relevant metrics are pool size, active workers, idle workers, and
		// queue depth. The thread cache variables are irrelevant and deliberately
		// omitted to avoid confusion.
		//
		// tpQueuedQueries delta is the key indicator: if it rose during the
		// scenario, incoming requests had to wait for a free worker thread —
		// the server-side equivalent of the app pool's WaitCount.
		deltaQueued := mysqlAfter.tpQueuedQueries - mysqlBefore.tpQueuedQueries
		fmt.Printf("  MySQL Thread Pool (pool_size=%d, thread_handling=%s):\n",
			mysqlBefore.tpSize, mysqlBefore.threadHandling)
		fmt.Printf("    Before: conn=%-5d running=%-4d pool_threads=%-4d pool_idle=%-4d queued=%d\n",
			mysqlBefore.threadsConnected, mysqlBefore.threadsRunning,
			mysqlBefore.tpThreads, mysqlBefore.tpIdleThreads, mysqlBefore.tpQueuedQueries)
		fmt.Printf("    After:  conn=%-5d running=%-4d pool_threads=%-4d pool_idle=%-4d queued=%d\n",
			mysqlAfter.threadsConnected, mysqlAfter.threadsRunning,
			mysqlAfter.tpThreads, mysqlAfter.tpIdleThreads, mysqlAfter.tpQueuedQueries)
		fmt.Printf("    New MySQL connections: %-6d  Queue depth increase: %d\n",
			deltaConns, deltaQueued)

		if psDeltaCount > 0 {
			fmt.Printf("    PS socket events: %-6d  Avg server-side accept latency: %dµs\n",
				psDeltaCount, psAvgConnNs/1000)
		}

		// Warn if the thread pool queue grew during the scenario — this means
		// requests were rejected from immediate execution and had to wait.
		// Unlike the app-side WaitCount (which is always recoverable), a
		// growing server-side queue can eventually cause client errors if
		// threadpool_max_transactions_limit is exceeded.
		if deltaQueued > 0 {
			fmt.Printf("\n  [!] Server thread pool queue grew by %d during the scenario.\n", deltaQueued)
			fmt.Printf("      Worker threads may be saturated. Consider increasing threadpool_size (currently %d).\n",
				mysqlBefore.tpSize)
		}
	} else {
		// One-thread-per-connection mode.
		// The thread cache hit rate is the primary server-side health indicator.
		fmt.Printf("  MySQL (thread_cache_size=%d, thread_handling=%s):\n",
			mysqlBefore.threadCacheSize, mysqlBefore.threadHandling)
		fmt.Printf("    Before: conn=%-5d running=%-4d cached=%-4d created=%d\n",
			mysqlBefore.threadsConnected, mysqlBefore.threadsRunning,
			mysqlBefore.threadsCached, mysqlBefore.threadsCreated)
		fmt.Printf("    After:  conn=%-5d running=%-4d cached=%-4d created=%d\n",
			mysqlAfter.threadsConnected, mysqlAfter.threadsRunning,
			mysqlAfter.threadsCached, mysqlAfter.threadsCreated)
		fmt.Printf("    New MySQL connections: %-6d  New threads spawned: %-4d  Cache hit: %.1f%%\n",
			deltaConns, deltaCreated, cacheHitPct)

		// Socket accept latency from performance_schema.
		// Only printed when new TCP connections were made during the scenario.
		// If MaxIdleConns was large enough that all workers reused idle connections,
		// psDeltaCount will be 0 — which itself is informative (no new OS-level
		// accept overhead at all).
		if psDeltaCount > 0 {
			fmt.Printf("    PS socket events: %-6d  Avg server-side accept latency: %dµs\n",
				psDeltaCount, psAvgConnNs/1000)
		}
		if deltaCreated > 0 && mysqlBefore.threadCacheSize > 0 {
			used := mysqlAfter.threadsCached
			pctFull := float64(used) / float64(mysqlBefore.threadCacheSize) * 100
			fmt.Printf("    Thread cache: %d/%d used (%.0f%%)\n",
				used, mysqlBefore.threadCacheSize, pctFull)
		}
	}

	// ── Actionable warnings ───────────────────────────────────────────────────
	if deltaWaitCount > 0 {
		fmt.Printf("\n  [!] App pool contention: %d ops queued for a connection (total wait %s).\n",
			deltaWaitCount, deltaWaitDur.Round(time.Millisecond))
		fmt.Printf("      Increase --maxOpenConns (currently %d) or reduce --workers (%d).\n",
			t.MaxOpenConns, workers)
	}
	// Only warn about thread cache in the model where thread_cache_size matters.
	if !usingThreadPool && cacheHitPct < 80 && deltaCreated > 0 {
		fmt.Printf("  [!] Low MySQL thread cache hit rate (%.1f%%).\n", cacheHitPct)
		fmt.Printf("      Increase thread_cache_size (currently %d) on the server.\n",
			mysqlBefore.threadCacheSize)
	}

	fmt.Println(bar)
}

// ─────────────────────────────────────────────────────────────────────────────
// Histograms
// ─────────────────────────────────────────────────────────────────────────────

// printHistograms renders ASCII frequency histograms for acquire and release
// latency after each scenario. Histograms reveal distribution shape — whether
// latency is unimodal (healthy), bimodal (pool pressure causing two distinct
// populations: fast reuse vs slow wait), or has a long tail (intermittent slow
// connections). This is not visible from percentiles alone.
func (t *ConnectionPoolTest) printHistograms(results []cpResult) {
	acq := make([]int64, 0, len(results))
	rel := make([]int64, 0, len(results))
	for _, r := range results {
		if r.err == nil {
			acq = append(acq, r.acquireNs)
			rel = append(rel, r.releaseNs)
		}
	}
	cpHistogram("Acquire", acq)
	cpHistogram("Release", rel)
}

// cpHistogram prints a horizontal bar chart with fixed logarithmic-ish buckets.
// The bucket boundaries are chosen to match typical connection latency ranges:
//   - Sub-100µs: pool reuse, no OS thread creation
//   - 100µs-1ms: pool reuse with some overhead (large pool, GC pause, etc.)
//   - 1ms-10ms: new TCP connection being served from the thread cache
//   - 10ms+: new TCP connection requiring a new OS thread, or network issues
//
// Bar lengths are normalised to the busiest bucket so that the chart fills the
// terminal width regardless of distribution shape (a single dominant bucket
// does not squash all other bars to zero).
func cpHistogram(label string, ns []int64) {
	type bucket struct {
		label  string
		lo, hi int64 // nanosecond boundaries [lo, hi)
	}
	buckets := []bucket{
		{"  <100µs", 0, 100_000},
		{"100-500µs", 100_000, 500_000},
		{"0.5-1ms  ", 500_000, 1_000_000},
		{"  1-5ms  ", 1_000_000, 5_000_000},
		{"  5-10ms ", 5_000_000, 10_000_000},
		{" 10-50ms ", 10_000_000, 50_000_000},
		{" 50-100ms", 50_000_000, 100_000_000},
		{"  >100ms ", 100_000_000, math.MaxInt64},
	}

	counts := make([]int64, len(buckets))
	for _, v := range ns {
		for i, b := range buckets {
			if v >= b.lo && v < b.hi {
				counts[i]++
				break
			}
		}
	}

	total := int64(len(ns))
	var maxCount int64
	for _, c := range counts {
		if c > maxCount {
			maxCount = c
		}
	}

	// width is the number of block characters (█ / ░) in the bar area.
	// 42 fits comfortably in an 80-column terminal alongside the label and
	// percentage columns.
	const width = 42
	sep := strings.Repeat("─", 70)
	fmt.Printf("\n  %s Latency Histogram (%d samples)\n", label, total)
	fmt.Println("  " + sep)
	for i, b := range buckets {
		c := counts[i]
		pct := 0.0
		if total > 0 {
			pct = float64(c) / float64(total) * 100
		}
		// Scale the bar relative to the busiest bucket, not to the total.
		// This prevents the most-common bucket from monopolising the entire
		// bar width and makes minority buckets visible.
		filled := 0
		if maxCount > 0 {
			filled = int(float64(c) / float64(maxCount) * float64(width))
		}
		bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
		fmt.Printf("  %-10s │%s│ %8d  %5.1f%%\n", b.label, bar, c, pct)
	}
	fmt.Println("  " + sep)
}

// ─────────────────────────────────────────────────────────────────────────────
// Header and CSV column definitions
// ─────────────────────────────────────────────────────────────────────────────

// printHeader prints the test configuration to stdout before the run starts.
// It also queries the current MySQL thread cache state so the operator can
// verify the server is in the expected baseline state before the test begins.
// Suppressed in CSV mode to keep stdout machine-parseable.
func (t *ConnectionPoolTest) printHeader(db *sql.DB) {
	if t.ReportCSV {
		return
	}
	const bar = "══════════════════════════════════════════════════════════════════════"
	fmt.Println(bar)
	fmt.Println("  MySQL Connection Pool Test")
	fmt.Printf("  URL:        %s\n", t.Parameters.Url)
	fmt.Printf("  App Pool:   MaxOpen=%-4d  MaxIdle=%-4d  MaxLifetime=%ds\n",
		t.MaxOpenConns, t.MaxIdleConns, t.ConnMaxLifetimeSec)

	mode := fmt.Sprintf("loop-based (%d loops)", t.Loops)
	if t.Duration > 0 {
		mode = fmt.Sprintf("time-based (%ds)", t.Duration)
	}
	ramp := ""
	if t.RampMode {
		ramp = " [ramp]"
	}
	fmt.Printf("  Mode:       %s  Workers=%d%s\n", mode, t.Workers, ramp)
	fmt.Printf("  Read:       %s | Write: %d bytes | ConnReuse: %d\n",
		t.PayloadSize, t.WriteSize, t.ConnReuse)
	fmt.Printf("  Warmup:     %d iters | ReportInterval: %ds\n",
		t.WarmupLoops, t.ReportInterval)

	ms := t.captureMySQLStats(db)
	if ms.threadHandling == "pool-of-threads" {
		fmt.Printf("  MySQL:      thread_handling=pool-of-threads  pool_size=%d  pool_threads=%d  pool_idle=%d  queued=%d\n",
			ms.tpSize, ms.tpThreads, ms.tpIdleThreads, ms.tpQueuedQueries)
	} else {
		fmt.Printf("  MySQL:      thread_handling=%s  thread_cache_size=%d  cached=%d  created=%d\n",
			ms.threadHandling, ms.threadCacheSize, ms.threadsCached, ms.threadsCreated)
	}
	fmt.Println(bar)
}

// printCSVHeaders prints comment lines that describe the two CSV row formats:
// "interval," rows from the live reporter and "summary," rows from the scenario
// report. Using a "#" prefix makes these lines ignorable by standard CSV tools
// (pandas, awk, etc.) while remaining readable for humans.
func (t *ConnectionPoolTest) printCSVHeaders() {
	if t.ReportInterval > 0 {
		fmt.Println("# interval,t_sec,ops,errors,tps,acq_avg_us,rel_avg_us,rd_avg_us,wr_avg_us,rd_kb_s,wr_kb_s")
	}
	// thread_handling identifies which model is active. Columns that follow it:
	//   cache_hit_pct    — meaningful only when thread_handling=one-thread-per-connection; 0.0 otherwise
	//   tp_queued_*      — meaningful only when thread_handling=pool-of-threads; 0 otherwise
	fmt.Println("# summary,workers,loops,duration_s,read_size,write_bytes,conn_reuse,elapsed_s," +
		"max_open,max_idle," +
		"ops,errors,tps,qps,rd_kb_s,wr_kb_s," +
		"acq_avg_ns,acq_p50_ns,acq_p95_ns,acq_p99_ns,acq_max_ns," +
		"rel_avg_ns,rel_p50_ns,rel_p95_ns,rel_p99_ns,rel_max_ns," +
		"rd_avg_ns,rd_p50_ns,rd_p95_ns,rd_p99_ns,rd_max_ns," +
		"wr_avg_ns,wr_p50_ns,wr_p95_ns,wr_p99_ns,wr_max_ns," +
		"pool_open,pool_in_use,pool_idle,pool_wait_count,pool_wait_ms," +
		"thread_handling,mysql_conn_before,mysql_created_before,mysql_conn_after," +
		"mysql_cached_before,mysql_cached_after,mysql_new_threads,thread_cache_size,cache_hit_pct," +
		"tp_queued_before,tp_queued_after,tp_queued_delta")
}

// ─────────────────────────────────────────────────────────────────────────────
// Query helper
// ─────────────────────────────────────────────────────────────────────────────

// readQuery returns the SQL statement used for the read phase.
// The query is chosen to produce a fixed, known-size result without touching
// any user table, so that result size is entirely determined by the test
// parameters and not by data distribution. REPEAT() executes purely in the
// MySQL expression engine — no storage I/O — so the measured latency reflects
// network transfer time and protocol overhead rather than disk access.
func (t *ConnectionPoolTest) readQuery() string {
	switch t.PayloadSize {
	case "medium":
		return "SELECT REPEAT('x', 1024) AS data"   // ~1 KB
	case "large":
		return "SELECT REPEAT('x', 65536) AS data"  // ~64 KB
	case "xlarge":
		return "SELECT REPEAT('x', 1048576) AS data" // ~1 MB
	default:
		return "SELECT @@hostname AS data" // ~10–50 B
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Progress bar
// ─────────────────────────────────────────────────────────────────────────────

// cpProgressBar writes an in-place progress bar to stdout by emitting a
// carriage return (\r) to reposition the cursor at the start of the current
// line rather than advancing to a new line. This overwrites the previous bar
// with the updated one, producing an animated effect without scrolling.
// The trailing spaces after the bar erase any remnant characters from a wider
// previous line (e.g. a longer elapsed time string).
func cpProgressBar(cur, total int64, elapsed time.Duration) {
	const width = 30
	pct := float64(cur) / float64(total)
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Printf("\r  [%s] %3.0f%%  %d/%d  %s   ",
		bar, pct*100, cur, total, elapsed.Round(time.Millisecond))
}

// ─────────────────────────────────────────────────────────────────────────────
// Statistics helpers
// ─────────────────────────────────────────────────────────────────────────────

// cpMean computes the arithmetic mean of a nanosecond latency slice.
// Returns 0 for an empty slice to avoid divide-by-zero panics.
func cpMean(vals []int64) float64 {
	if len(vals) == 0 {
		return 0
	}
	var sum int64
	for _, v := range vals {
		sum += v
	}
	return float64(sum) / float64(len(vals))
}

// cpPct returns the p-th percentile value from a pre-sorted int64 slice
// using the "nearest rank" method (ceiling of n×p/100 − 1). The slice must
// be sorted ascending before calling this function. Returns 0 for an empty
// slice. p must be in the range [0, 100].
func cpPct(sorted []int64, p int) int64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(n*p)/100)) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
