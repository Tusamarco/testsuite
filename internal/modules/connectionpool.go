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

// ConnectionPoolTest measures connection efficiency and pool interaction between
// the application layer (sql.DB pool) and the MySQL server thread cache.
type ConnectionPoolTest struct {
	*TestBase
	Workers            int
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetimeSec int
	PayloadSize        string
	RampMode           bool
	WarmupLoops        int
}

type cpOpResult struct {
	acquireNs int64
	queryNs   int64
	releaseNs int64
	bytesRead int64
	err       error
}

type mysqlThreadStats struct {
	threadsConnected int64
	threadsRunning   int64
	threadsCached    int64
	threadsCreated   int64
	connections      int64
	threadCacheSize  int64
}

type appPoolSnapshot struct {
	openConnections int
	inUse           int
	idle            int
	waitCount       int64
	waitDuration    time.Duration
	maxIdleClosed   int64
	maxLifeClosed   int64
}

func NewConnectionPoolTest(params config.Params) *ConnectionPoolTest {
	tb := NewTestBase(params)
	return &ConnectionPoolTest{
		TestBase:           tb,
		Workers:            params.Workers,
		MaxOpenConns:       params.MaxOpenConns,
		MaxIdleConns:       params.MaxIdleConns,
		ConnMaxLifetimeSec: params.ConnMaxLifetimeSec,
		PayloadSize:        params.PayloadSize,
		RampMode:           params.RampWorkers,
		WarmupLoops:        params.WarmupLoops,
	}
}

func (t *ConnectionPoolTest) Run() {
	t.Init()

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
	db.SetMaxOpenConns(t.MaxOpenConns)
	db.SetMaxIdleConns(t.MaxIdleConns)
	if t.ConnMaxLifetimeSec > 0 {
		db.SetConnMaxLifetime(time.Duration(t.ConnMaxLifetimeSec) * time.Second)
	}

	t.printHeader(db)

	if t.WarmupLoops > 0 {
		t.warmup(db)
	}

	workerLevels := t.workerLevels()

	if t.ReportCSV {
		fmt.Println("workers,loops,payload_size,max_open_conns,max_idle_conns," +
			"avg_acquire_ns,p50_acquire_ns,p95_acquire_ns,p99_acquire_ns,max_acquire_ns," +
			"avg_query_ns,p50_query_ns,p95_query_ns,p99_query_ns,max_query_ns," +
			"avg_total_ns,throughput_qps,bytes_per_sec," +
			"app_open,app_in_use,app_idle,app_wait_count,app_wait_ms," +
			"mysql_connected_before,mysql_cached_before,mysql_created_before," +
			"mysql_connected_after,mysql_cached_after,mysql_created_after," +
			"new_threads_created,thread_cache_hit_pct,errors")
	}

	for _, w := range workerLevels {
		t.runScenario(db, w)
	}

	os.Exit(0)
}

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

// payloadQuery returns the SQL query for the configured payload size.
// Sizes: small=~10B, medium=~1KB, large=~64KB, xlarge=~1MB
func (t *ConnectionPoolTest) payloadQuery() string {
	switch t.PayloadSize {
	case "medium":
		return "SELECT REPEAT('x', 1024) AS data"
	case "large":
		return "SELECT REPEAT('x', 65536) AS data"
	case "xlarge":
		return "SELECT REPEAT('x', 1048576) AS data"
	default:
		return "SELECT @@hostname AS data"
	}
}

func (t *ConnectionPoolTest) warmup(db *sql.DB) {
	if !t.ReportCSV {
		fmt.Printf("Warming up pool (%d iterations)...\n", t.WarmupLoops)
	}
	ctx := context.Background()
	q := t.payloadQuery()
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

func (t *ConnectionPoolTest) captureMySQLStats(db *sql.DB) mysqlThreadStats {
	stats := mysqlThreadStats{}
	ctx := context.Background()

	rows, err := db.QueryContext(ctx,
		"SELECT variable_name, variable_value "+
			"FROM performance_schema.global_status "+
			"WHERE variable_name IN "+
			"('Threads_connected','Threads_running','Threads_cached','Threads_created','Connections')")
	if err != nil {
		// Fallback: some configurations restrict performance_schema access
		rows, err = db.QueryContext(ctx,
			"SHOW GLOBAL STATUS WHERE Variable_name IN "+
				"('Threads_connected','Threads_running','Threads_cached','Threads_created','Connections')")
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
		case "Threads_cached":
			stats.threadsCached = val
		case "Threads_created":
			stats.threadsCreated = val
		case "Connections":
			stats.connections = val
		}
	}

	db.QueryRowContext(ctx, "SELECT @@thread_cache_size").Scan(&stats.threadCacheSize) //nolint
	return stats
}

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

func (t *ConnectionPoolTest) runScenario(db *sql.DB, workers int) {
	query := t.payloadQuery()
	totalOps := workers * t.Loops

	mysqlBefore := t.captureMySQLStats(db)
	appBefore := snapshotAppPool(db)

	resultsCh := make(chan cpOpResult, totalOps)
	var wg sync.WaitGroup

	// Print progress every ~10% of total ops, minimum every 100, maximum every 10_000.
	progressEvery := int64(totalOps / 10)
	if progressEvery < 100 {
		progressEvery = 100
	}
	if progressEvery > 10_000 {
		progressEvery = 10_000
	}
	var opCounter int64

	startTime := time.Now()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := context.Background()
			for i := 0; i < t.Loops; i++ {
				t1 := time.Now()
				conn, err := db.Conn(ctx)
				t2 := time.Now()
				if err != nil {
					resultsCh <- cpOpResult{err: err}
					atomic.AddInt64(&opCounter, 1)
					continue
				}

				var data string
				qErr := conn.QueryRowContext(ctx, query).Scan(&data)
				t3 := time.Now()

				conn.Close()
				t4 := time.Now()

				res := cpOpResult{
					acquireNs: t2.Sub(t1).Nanoseconds(),
					queryNs:   t3.Sub(t2).Nanoseconds(),
					releaseNs: t4.Sub(t3).Nanoseconds(),
					bytesRead: int64(len(data)),
				}
				if qErr != nil {
					res.err = qErr
				}
				resultsCh <- res

				cur := atomic.AddInt64(&opCounter, 1)
				if !t.ReportCSV && cur%progressEvery == 0 {
					cpProgressBar(cur, int64(totalOps), time.Since(startTime))
				}

				if t.Sleep > 0 {
					time.Sleep(time.Duration(t.Sleep) * time.Millisecond)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(startTime)
	if !t.ReportCSV {
		cpProgressBar(int64(totalOps), int64(totalOps), elapsed)
		fmt.Println()
	}
	close(resultsCh)

	mysqlAfter := t.captureMySQLStats(db)
	appAfter := snapshotAppPool(db)

	var results []cpOpResult
	for r := range resultsCh {
		results = append(results, r)
	}

	t.printScenarioReport(workers, results, elapsed, mysqlBefore, mysqlAfter, appBefore, appAfter)
}

func (t *ConnectionPoolTest) printScenarioReport(
	workers int,
	results []cpOpResult,
	elapsed time.Duration,
	mysqlBefore, mysqlAfter mysqlThreadStats,
	appBefore, appAfter appPoolSnapshot,
) {
	var valid []cpOpResult
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

	acquire := make([]int64, n)
	query := make([]int64, n)
	total := make([]int64, n)
	var totalBytes int64

	for i, r := range valid {
		acquire[i] = r.acquireNs
		query[i] = r.queryNs
		total[i] = r.acquireNs + r.queryNs + r.releaseNs
		totalBytes += r.bytesRead
	}

	sort.Slice(acquire, func(i, j int) bool { return acquire[i] < acquire[j] })
	sort.Slice(query, func(i, j int) bool { return query[i] < query[j] })
	sort.Slice(total, func(i, j int) bool { return total[i] < total[j] })

	qps := float64(n) / elapsed.Seconds()
	bps := float64(totalBytes) / elapsed.Seconds()

	deltaConns := mysqlAfter.connections - mysqlBefore.connections
	deltaCreated := mysqlAfter.threadsCreated - mysqlBefore.threadsCreated
	var cacheHitPct float64
	if deltaConns > 0 {
		cacheHitPct = float64(deltaConns-deltaCreated) / float64(deltaConns) * 100
		if cacheHitPct < 0 {
			cacheHitPct = 0
		}
	}

	deltaWaitCount := appAfter.waitCount - appBefore.waitCount
	deltaWaitDur := appAfter.waitDuration - appBefore.waitDuration

	if t.ReportCSV {
		fmt.Printf("%d,%d,%s,%d,%d,%.0f,%d,%d,%d,%d,%.0f,%d,%d,%d,%d,%.0f,%.2f,%.0f,%d,%d,%d,%d,%.0f,%d,%d,%d,%d,%d,%d,%d,%.1f,%d\n",
			workers, t.Loops, t.PayloadSize,
			t.MaxOpenConns, t.MaxIdleConns,
			cpMean(acquire), cpPct(acquire, 50), cpPct(acquire, 95), cpPct(acquire, 99), acquire[n-1],
			cpMean(query), cpPct(query, 50), cpPct(query, 95), cpPct(query, 99), query[n-1],
			cpMean(total), qps, bps,
			appAfter.openConnections, appAfter.inUse, appAfter.idle, deltaWaitCount, deltaWaitDur.Milliseconds(),
			mysqlBefore.threadsConnected, mysqlBefore.threadsCached, mysqlBefore.threadsCreated,
			mysqlAfter.threadsConnected, mysqlAfter.threadsCached, mysqlAfter.threadsCreated,
			deltaCreated, cacheHitPct,
			errCount,
		)
		return
	}

	bar := "══════════════════════════════════════════════════════════════════════"
	sep := "──────────────────────────────────────────────────────────────────────"
	fmt.Println()
	fmt.Println(bar)
	fmt.Printf("  Workers: %-4d | Loops: %-6d | Payload: %-8s | Duration: %s\n",
		workers, t.Loops, t.PayloadSize, elapsed.Round(time.Millisecond))
	fmt.Printf("  App Pool: MaxOpen=%-4d MaxIdle=%-4d  Operations: %-6d Errors: %d\n",
		t.MaxOpenConns, t.MaxIdleConns, n, errCount)
	fmt.Printf("  Throughput: %.1f QPS | %.1f KB/s\n", qps, bps/1024)
	fmt.Println(sep)

	fmt.Printf("  %-12s  %12s  %12s  %12s  %12s  %12s\n",
		"Phase", "Avg (ns)", "P50 (ns)", "P95 (ns)", "P99 (ns)", "Max (ns)")
	fmt.Printf("  %-12s  %12.0f  %12d  %12d  %12d  %12d\n",
		"Acquire", cpMean(acquire), cpPct(acquire, 50), cpPct(acquire, 95), cpPct(acquire, 99), acquire[n-1])
	fmt.Printf("  %-12s  %12.0f  %12d  %12d  %12d  %12d\n",
		"Query", cpMean(query), cpPct(query, 50), cpPct(query, 95), cpPct(query, 99), query[n-1])
	fmt.Printf("  %-12s  %12.0f  %12d  %12d  %12d  %12d\n",
		"Round-trip", cpMean(total), cpPct(total, 50), cpPct(total, 95), cpPct(total, 99), total[n-1])

	fmt.Println(sep)
	fmt.Println("  App Pool (sql.DB) — end-of-scenario snapshot:")
	fmt.Printf("    OpenConns=%-4d  InUse=%-4d  Idle=%-4d\n",
		appAfter.openConnections, appAfter.inUse, appAfter.idle)
	fmt.Printf("    WaitCount(delta)=%-6d  WaitDuration(delta)=%s\n",
		deltaWaitCount, deltaWaitDur.Round(time.Millisecond))
	fmt.Printf("    MaxIdleClosed(delta)=%-4d  MaxLifetimeClosed(delta)=%d\n",
		appAfter.maxIdleClosed-appBefore.maxIdleClosed,
		appAfter.maxLifeClosed-appBefore.maxLifeClosed)

	fmt.Println(sep)
	fmt.Printf("  MySQL Server (thread_cache_size=%d):\n", mysqlBefore.threadCacheSize)
	fmt.Printf("    Before: connected=%-4d  running=%-4d  cached=%-4d  created=%d\n",
		mysqlBefore.threadsConnected, mysqlBefore.threadsRunning,
		mysqlBefore.threadsCached, mysqlBefore.threadsCreated)
	fmt.Printf("    After:  connected=%-4d  running=%-4d  cached=%-4d  created=%d\n",
		mysqlAfter.threadsConnected, mysqlAfter.threadsRunning,
		mysqlAfter.threadsCached, mysqlAfter.threadsCreated)
	fmt.Printf("    New connections to MySQL: %-6d  New threads spawned: %-6d  Cache hit rate: %.1f%%\n",
		deltaConns, deltaCreated, cacheHitPct)

	if deltaCreated > 0 && mysqlBefore.threadCacheSize > 0 {
		used := mysqlAfter.threadsCached
		pctFull := float64(used) / float64(mysqlBefore.threadCacheSize) * 100
		fmt.Printf("    Thread cache utilisation: %d/%d (%.0f%%)\n",
			used, mysqlBefore.threadCacheSize, pctFull)
	}

	if deltaWaitCount > 0 {
		fmt.Printf("\n  [!] App pool contention: %d goroutines waited (total %s).\n",
			deltaWaitCount, deltaWaitDur.Round(time.Millisecond))
		fmt.Printf("      Consider increasing --maxOpenConns (currently %d).\n", t.MaxOpenConns)
	}
	if cacheHitPct < 80 && deltaCreated > 0 {
		fmt.Printf("  [!] Low MySQL thread cache hit rate (%.1f%%). Consider increasing thread_cache_size=%d.\n",
			cacheHitPct, mysqlBefore.threadCacheSize)
	}
	fmt.Println(bar)
}

func (t *ConnectionPoolTest) printHeader(db *sql.DB) {
	if t.ReportCSV {
		return
	}
	bar := "══════════════════════════════════════════════════════════════════════"
	fmt.Println(bar)
	fmt.Println("  MySQL Connection Pool Test")
	fmt.Printf("  URL:        %s\n", t.Parameters.Url)
	fmt.Printf("  App Pool:   MaxOpenConns=%-4d  MaxIdleConns=%-4d  MaxLifetime=%ds\n",
		t.MaxOpenConns, t.MaxIdleConns, t.ConnMaxLifetimeSec)
	fmt.Printf("  Workers:    %d%s\n", t.Workers, func() string {
		if t.RampMode {
			return " (ramp mode)"
		}
		return ""
	}())
	fmt.Printf("  Loops:      %d per worker\n", t.Loops)
	fmt.Printf("  Payload:    %s\n", t.PayloadSize)
	fmt.Printf("  Warmup:     %d\n", t.WarmupLoops)

	mysqlStats := t.captureMySQLStats(db)
	fmt.Printf("  MySQL:      thread_cache_size=%d  currently cached=%d  created=%d\n",
		mysqlStats.threadCacheSize, mysqlStats.threadsCached, mysqlStats.threadsCreated)
	fmt.Println(bar)
}

// cpProgressBar prints an in-place progress bar to stdout, overwriting the current line.
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

// cpMean returns the arithmetic mean of a slice of int64.
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

// cpPct returns the p-th percentile from a pre-sorted slice (0–100).
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
