# MySQL Test Suite

A collection of Go tools for testing MySQL connectivity, connection pool behaviour, replication lag, and bulk data generation.

## Modules

| Module | Flag | Purpose |
|---|---|---|
| `connectionpool` | `--module connectionpool` | Measure connection acquisition latency, query throughput, and the interaction between the app-level pool (`sql.DB`) and the MySQL server thread cache |
| `staleread` | `--module staleread` | Detect replication lag between a writer and a reader node |
| `datagen` | `--module datagen` | Bulk-load realistic European person/address data (2 M+ rows) |

---

## Building

```bash
go build -o testsuite ./cmd/testsuite
```

Requires **Go 1.21+**.

---

## Common flags

These flags apply to every module.

| Flag | Default | Description |
|---|---|---|
| `--url` | | MySQL address as `host:port` |
| `--url-read` | | Reader node address (staleread only) |
| `--user` | `app_test` | Database user |
| `--password` | `test` | Database password |
| `--schema` | `mysql` | Default schema / database |
| `--loops` | `50` | Iterations per worker (loop-based mode) |
| `--sleep` | `0` | Milliseconds to sleep between iterations |
| `--verbose` | `false` | Extended per-iteration output |
| `--summary` | `false` | Print summary statistics at the end |
| `--reportCSV` | `false` | Machine-readable CSV output |

---

## Connection Pool module

### What it measures

The test decomposes every database interaction into three independently timed phases:

| Phase | What it captures |
|---|---|
| **Acquire** | `db.Conn()` — wall time from the call until a connection is ready. Includes any queue wait when all pool slots are busy. This is where pool pressure is visible. |
| **Read / Write** | Time spent inside `QueryRowContext` / `ExecContext`. Isolates network RTT + server execution from connection overhead. |
| **Release** | `conn.Close()` — returning the connection to the pool. Normally microseconds; a spike here indicates Go runtime scheduling problems. |

After each scenario the module queries both `sql.DB.Stats()` and MySQL status variables (including `performance_schema.events_waits_summary_global_by_event_name`) to surface:

- App pool: `OpenConns`, `InUse`, `Idle`, `WaitCount`, `WaitDuration`, `MaxIdleClosed`, `MaxLifetimeClosed`
- MySQL thread cache: `Threads_connected`, `Threads_running`, `Threads_cached`, `Threads_created`, `thread_cache_size`
- Thread-cache hit rate: `(ΔConnections − ΔThreads_created) / ΔConnections × 100`
- Server-side connection accept latency from `performance_schema` (picoseconds converted to µs)

Finally, two ASCII histograms show the **distribution** of Acquire and Release latency, which reveals bimodal distributions (fast pool hits vs slow waits) that percentiles alone cannot expose.

---

### All flags

| Flag | Default | Description |
|---|---|---|
| `--workers` | `8` | Concurrent goroutines. Each goroutine runs independently and sends operations to the same `sql.DB` pool. |
| `--loops` | `50` | Operations per worker (ignored when `--duration` is set). |
| `--duration` | `0` | Run duration in **seconds**. When > 0, workers execute continuously until the deadline, then stop cleanly. Use instead of `--loops` for steady-state tests. |
| `--maxOpenConns` | `10` | `sql.DB` `SetMaxOpenConns` — hard cap on open connections. When `workers > maxOpenConns`, goroutines queue inside the pool and `WaitCount` rises. |
| `--maxIdleConns` | `5` | `sql.DB` `SetMaxIdleConns` — connections kept alive between operations. `0` forces a new TCP connection + MySQL auth for every single operation. |
| `--connMaxLifetimeSec` | `0` | `sql.DB` `SetConnMaxLifetime` in seconds. `0` = unlimited. When set, the pool evicts and recreates connections after this many seconds, which increases `Threads_created` on the server. |
| `--payloadSize` | `small` | Read query data volume: `small` (~10 B, `SELECT @@hostname`), `medium` (~1 KB), `large` (~64 KB), `xlarge` (~1 MB). Controls how much of round-trip latency is data transfer vs connection overhead. |
| `--writeSize` | `0` | Write payload **bytes** per operation. `0` = read-only. When > 0, a scratch table `_cptest_scratch` is created in `--schema`, and each worker UPDATEs its own dedicated row each cycle. |
| `--connReuse` | `1` | Number of read (+ optional write) SQL statements executed on a single acquired connection before it is released. `1` = pure connection overhead measurement. Higher values model ORMs that reuse connections across multiple queries per request. |
| `--rampWorkers` | `false` | Run a series of scenarios at increasing concurrency: 1 → 4 → 16 → 32 → 64 → 128 → 256 → … → `--workers`. Produces a scaling curve showing where throughput saturates and latency spikes. |
| `--warmupLoops` | `5` | Sequential pre-measurement cycles to prime the pool and MySQL thread cache before timing starts. |
| `--reportInterval` | `5` | Sysbench-style live stats interval in seconds. `0` = disable. When active, shows per-interval ops, TPS, and average latency while the test runs. Replaces the progress bar. |
| `--reportCSV` | `false` | Machine-readable output. Interval lines are prefixed `interval,`; summary lines are prefixed `summary,`. Both can coexist in one redirected file and be split by `grep`. |
| `--sleep` | `0` | Milliseconds each worker sleeps between operations. Simulates think-time / pacing and reduces QPS to a controlled rate. |

---

### Output explained

**Interval output** (during the run, every `--reportInterval` seconds):

```
  t(s)   ops      errors  tps        acq_avg_µs  rel_avg_µs  rd_avg_µs   wr_avg_µs   rd_KB/s    wr_KB/s
  5      6241     0       1248.2     41.3         6.1         98.4        0.0         155.12     0.00
  10     6390     0       1278.0     39.8         5.9         97.1        0.0         158.94     0.00
```

**Scenario summary** (after each scenario):

```
══════════════════════════════════════════════════════════════════════
  Workers: 32    | Ops: 6400     | Duration: 1.243s
  Read: small    | Write: 0       bytes | ConnReuse: 1 queries/conn
  App Pool: MaxOpen=32   MaxIdle=16   | Errors: 0
  Throughput: 5151.5 TPS | 5151.5 QPS | 71.20 KB/s read | 0.00 KB/s write
──────────────────────────────────────────────────────────────────────
  Phase          Avg (µs)     P50 (µs)     P95 (µs)     P99 (µs)     Max (µs)
  Acquire               42           28          113          288         1842
  Read                 139          127          222          389         2103
  Release                8            6           14           28          144
  Round-trip           184          162          348          701         3981
──────────────────────────────────────────────────────────────────────
  App Pool (sql.DB):
    OpenConns=32    InUse=0     Idle=16
    WaitCount(Δ)=0            WaitDuration(Δ)=0s
    MaxIdleClosed(Δ)=0        MaxLifetimeClosed(Δ)=0
──────────────────────────────────────────────────────────────────────
  MySQL (thread_cache_size=32):
    Before: conn=3     running=1    cached=8     created=112
    After:  conn=3     running=1    cached=16    created=112
    New MySQL connections: 6400    New threads spawned: 0      Cache hit: 100.0%
    PS socket events: 6400    Avg server-side accept latency: 18µs
══════════════════════════════════════════════════════════════════════
```

**Acquire latency histogram** (after the summary):

```
  Acquire Latency Histogram (6400 samples)
  ──────────────────────────────────────────────────────────────────────
    <100µs   │████████████████████████████████████████│     5312   83.0%
  100-500µs │█████████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│      960   15.0%
  0.5-1ms   │█░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│       96    1.5%
    1-5ms   │░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░│       32    0.5%
```

**What to look for:**

| Observation | Likely cause | Action |
|---|---|---|
| Acquire P99 >> Avg | Pool slots exhausted; goroutines queued | Increase `--maxOpenConns` |
| `WaitCount > 0` | Pool exhaustion confirmed | Increase `--maxOpenConns` or reduce `--workers` |
| Acquire bimodal histogram | Two populations: fast idle reuse + slow new-connection path | Increase `--maxIdleConns` |
| `New threads spawned > 0` | MySQL thread cache unable to absorb connection rate | Increase `thread_cache_size` on the server |
| `Cache hit < 80%` | Same as above, with a severity indicator | Same |
| `MaxLifetimeClosed > 0` | Pool recycling connections due to `connMaxLifetimeSec` | Review lifetime setting vs workload pattern |
| Release P99 spike | Go runtime scheduling jitter | Usually harmless; may indicate GC pressure |

---

### Scenarios

#### 1 — Baseline: establish a reference latency

Run a single worker with read-only small payload. This gives you the raw cost of one full connection cycle on your infrastructure, with no contention of any kind.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 1 --loops 500 \
  --maxOpenConns 5 --maxIdleConns 2 \
  --payloadSize small \
  --warmupLoops 10
```

Save this result as your baseline before varying any other parameter. The Acquire P50 here is the floor — it cannot get lower because there is no contention.

---

#### 2 — Concurrency ramp: find the saturation point

Ramp from 1 to 128 workers in steps, keeping the pool size matched to the worker count. Compare TPS and P99 acquire time at each level to find where throughput stops scaling.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 128 --rampWorkers \
  --loops 200 --payloadSize small \
  --maxOpenConns 128 --maxIdleConns 64 \
  --reportInterval 0
```

Collect CSV for graphing:

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 128 --rampWorkers \
  --loops 200 --payloadSize small \
  --maxOpenConns 128 --maxIdleConns 64 \
  --reportCSV > ramp.csv

# Extract summary rows only
grep '^summary,' ramp.csv | cut -d, -f2,13,18,19,20,21
# workers, tps, acq_avg_ns, acq_p50_ns, acq_p95_ns, acq_p99_ns
```

---

#### 3 — Pool pressure: force goroutine queuing

Set `--workers` larger than `--maxOpenConns` so goroutines must wait for a free pool slot. `WaitCount` and `WaitDuration` will rise; Acquire P99 will spike. This is the most common misconfiguration in production.

```bash
# 32 workers but only 10 pool slots — 22 goroutines will queue at peak
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 300 \
  --maxOpenConns 10 --maxIdleConns 5 \
  --payloadSize small \
  --reportInterval 5
```

Expected warning:

```
  [!] App pool contention: 847 ops queued for a connection (total wait 4.231s).
      Increase --maxOpenConns (currently 10) or reduce --workers (32).
```

Then fix it and compare:

```bash
# Pool sized to match workers — contention disappears
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 300 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --payloadSize small \
  --reportInterval 5
```

---

#### 4 — Time-based steady-state test

Use `--duration` instead of `--loops` to run for a fixed wall time. Useful for measuring stable throughput without waiting for a predetermined operation count to complete.

```bash
# 60-second run with 64 workers, live stats every 10 seconds
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 64 --duration 60 \
  --maxOpenConns 64 --maxIdleConns 32 \
  --payloadSize small \
  --reportInterval 10
```

```bash
# Same run, CSV output for all intervals and the final summary
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 64 --duration 60 \
  --maxOpenConns 64 --maxIdleConns 32 \
  --payloadSize small --reportInterval 10 \
  --reportCSV > steady_state.csv

grep '^interval,' steady_state.csv   # live per-interval rows
grep '^summary,'  steady_state.csv   # final aggregate row
```

---

#### 5 — Read payload size: isolate data-transfer cost

Fix concurrency and vary `--payloadSize`. The difference between small and xlarge Acquire times should be near zero (the connection cost is identical), while Read latency will grow proportionally to transferred bytes.

```bash
for size in small medium large xlarge; do
  ./testsuite --module connectionpool \
    --url 127.0.0.1:3306 --user root --password secret \
    --workers 16 --loops 200 \
    --maxOpenConns 16 --maxIdleConns 8 \
    --payloadSize $size \
    --reportCSV
done > payload_sweep.csv
```

---

#### 6 — Write test: mixed read/write workload

Add `--writeSize` to include an UPDATE in each connection cycle. The scratch table `_cptest_scratch` is created automatically in the schema specified by `--schema`.

```bash
# 16 workers, 1 KB read + 4 KB write per connection, 200 loops
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema mydb \
  --workers 16 --loops 200 \
  --maxOpenConns 16 --maxIdleConns 8 \
  --payloadSize medium --writeSize 4096 \
  --reportInterval 5
```

Compare write latency between platforms (e.g. local SSD vs network storage):

```bash
# Platform A
./testsuite --module connectionpool \
  --url platform-a:3306 --user root --password secret --schema test \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --payloadSize small --writeSize 1024 \
  --reportCSV > platform_a.csv

# Platform B
./testsuite --module connectionpool \
  --url platform-b:3306 --user root --password secret --schema test \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --payloadSize small --writeSize 1024 \
  --reportCSV > platform_b.csv

# Compare: workers, tps, read p99, write p99, cache_hit_pct
grep '^summary,' platform_a.csv platform_b.csv \
  | cut -d, -f2,13,29,34,49
```

---

#### 7 — Connection reuse: amortise connection overhead

Use `--connReuse` to run multiple SQL statements on a single acquired connection before releasing it. This models ORMs and query layers that batch several queries per connection checkout (a typical web request handler).

```bash
# connReuse=1: baseline — new connection for every single query
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --payloadSize small --connReuse 1

# connReuse=10: connection cost amortised across 10 queries
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --payloadSize small --connReuse 10
```

With `connReuse=10`, Acquire latency per query drops by ~10x and QPS rises significantly, while Acquire P99 per cycle stays the same. The difference shows how much your workload gains from connection reuse.

Combined with writes:

```bash
# 5 reads + 5 writes per connection acquisition
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema mydb \
  --workers 16 --loops 300 \
  --maxOpenConns 16 --maxIdleConns 8 \
  --payloadSize small --writeSize 512 --connReuse 5 \
  --reportInterval 5
```

---

#### 8 — Thread cache pressure: force OS thread creation

Set `--maxIdleConns 0` to force the pool to close connections after every operation, and use a high worker count. Every acquire creates a fresh TCP connection to MySQL, which must be served by a MySQL OS thread. If `thread_cache_size` is too small, MySQL spawns new OS threads on every connection, which is far more expensive than thread-cache reuse.

```bash
# No idle connections — every acquire is a fresh TCP + auth + OS thread
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 64 --loops 200 \
  --maxOpenConns 64 --maxIdleConns 0 \
  --payloadSize small
```

Expected warning when `thread_cache_size` is small:

```
  [!] Low MySQL thread cache hit rate (34.2%).
      Increase thread_cache_size (currently 8) on the server.
```

After increasing `thread_cache_size` on the server:

```sql
SET GLOBAL thread_cache_size = 128;
```

Re-run to confirm the hit rate recovers to 100%.

---

#### 9 — Connection lifetime: impact of pool recycling

`--connMaxLifetimeSec` forces the pool to evict and recreate connections after a fixed age. Combined with high concurrency, this generates a constant stream of new MySQL connections even when idle connections are available — stressing both the thread cache and the auth path.

```bash
# Unlimited lifetime (default) — pool reuses connections aggressively
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --connMaxLifetimeSec 0 --payloadSize small

# 5-second lifetime — forces periodic reconnects
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 500 \
  --maxOpenConns 32 --maxIdleConns 16 \
  --connMaxLifetimeSec 5 --payloadSize small
```

Compare `mysql_new_threads` and `cache_hit_pct` in the two runs. A low hit rate with `connMaxLifetimeSec=5` but 100% without it confirms that lifetime-based recycling is the source of thread-cache pressure.

---

#### 10 — Paced load: controlled QPS with sleep

Use `--sleep` to insert a think-time between operations, simulating application-side pacing or limiting the tool to a specific target QPS without reducing worker count.

```bash
# 64 workers, 100ms sleep — approximately 640 ops/sec max
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 64 --duration 60 \
  --maxOpenConns 64 --maxIdleConns 32 \
  --payloadSize small --sleep 100 \
  --reportInterval 10
```

Useful for load-testing at a specific rate before a production deployment.

---

#### 11 — ProxySQL / HAProxy comparison

Point `--url` at a proxy instead of MySQL directly. The acquire time now includes proxy routing overhead. Compare the CSV summaries from direct and proxy runs to quantify proxy cost.

```bash
# Direct to MySQL primary
./testsuite --module connectionpool \
  --url mysql-primary:3306 --user root --password secret \
  --workers 32 --loops 300 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 16 \
  --reportCSV > direct.csv

# Via ProxySQL
./testsuite --module connectionpool \
  --url proxysql:6033 --user root --password secret \
  --workers 32 --loops 300 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 16 \
  --reportCSV > proxy.csv

# Compare acquire latency
grep '^summary,' direct.csv | cut -d, -f17,18,19,20   # acq avg/p50/p95/p99 ns
grep '^summary,' proxy.csv  | cut -d, -f17,18,19,20
```

---

#### 12 — Full combined test: read + write + ramp + live output

A realistic multi-scenario test that covers everything simultaneously.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema mydb \
  --workers 256 --rampWorkers \
  --duration 30 \
  --maxOpenConns 256 --maxIdleConns 128 \
  --payloadSize medium --writeSize 1024 \
  --connReuse 3 \
  --warmupLoops 20 \
  --reportInterval 10
```

What this shows:
- Each ramp level runs for 30 seconds at that worker count
- `connReuse=3` means 3 reads + 3 writes per acquire → 6 SQL statements per cycle
- Live per-10s intervals let you observe warmup transients at each concurrency level
- The histogram at each level reveals whether the distribution is clean or bimodal

---

#### 13 — Cross-platform comparison template

Use this script to capture identical runs against two platforms and diff the results.

```bash
#!/bin/bash
COMMON="--workers 64 --loops 500 --payloadSize small --writeSize 1024 \
        --connReuse 5 --maxOpenConns 64 --maxIdleConns 32 \
        --warmupLoops 10 --reportCSV"

./testsuite --module connectionpool --url platform-a:3306 \
  --user root --password secret --schema bench $COMMON > a.csv

./testsuite --module connectionpool --url platform-b:3306 \
  --user root --password secret --schema bench $COMMON > b.csv

echo "=== Platform A ==="
grep '^summary,' a.csv | awk -F, '{
  printf "TPS=%.1f  acq_p99=%dµs  wr_p99=%dµs  cache_hit=%.1f%%\n",
    $13, $20/1000, $35/1000, $49}'

echo "=== Platform B ==="
grep '^summary,' b.csv | awk -F, '{
  printf "TPS=%.1f  acq_p99=%dµs  wr_p99=%dµs  cache_hit=%.1f%%\n",
    $13, $20/1000, $35/1000, $49}'
```

---

### CSV format reference

All values in the `summary,` row are in nanoseconds except where noted.

```
summary,
  workers, loops, duration_s, read_size, write_bytes, conn_reuse, elapsed_s,
  max_open, max_idle,
  ops, errors, tps, qps, rd_kb_s, wr_kb_s,
  acq_avg_ns, acq_p50_ns, acq_p95_ns, acq_p99_ns, acq_max_ns,
  rel_avg_ns, rel_p50_ns, rel_p95_ns, rel_p99_ns, rel_max_ns,
  rd_avg_ns,  rd_p50_ns,  rd_p95_ns,  rd_p99_ns,  rd_max_ns,
  wr_avg_ns,  wr_p50_ns,  wr_p95_ns,  wr_p99_ns,  wr_max_ns,
  pool_open, pool_in_use, pool_idle, pool_wait_count, pool_wait_ms,
  mysql_conn_before, mysql_cached_before, mysql_created_before,
  mysql_conn_after,  mysql_cached_after,  mysql_created_after,
  mysql_new_threads, cache_hit_pct, ps_conn_avg_ns
```

`interval,` rows (emitted every `--reportInterval` seconds):

```
interval,
  t_sec, ops, errors, tps,
  acq_avg_us, rel_avg_us, rd_avg_us, wr_avg_us,
  rd_kb_s, wr_kb_s
```

---

## Stale Read module

Tests replication lag between a writer and a reader. Writes a row to the writer then immediately reads it back from the reader, measuring how long the reader takes to catch up.

```bash
./testsuite --module staleread \
  --url       192.168.4.22:3306 \
  --url-read  192.168.4.23:3306 \
  --user app_test --password secret --schema test \
  --loops 10000 --sleep 2000 --summary
```

| Flag | Description |
|---|---|
| `--url` | Writer node |
| `--url-read` | Reader node (must differ from writer) |
| `--loops` | Number of write/read cycles |
| `--sleep` | Milliseconds to wait between cycles |
| `--rowsNumber` | Rows pre-loaded into the test table (default 10 000) |

---

## DataGen module

Seeds reference tables (continents, countries, cities, streets, names) then generates addresses and users in parallel using real European data.

```bash
# Default: 2 M users, 500 K addresses, 8 workers
./testsuite --module datagen \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema bobo \
  --rowsNumber 2000000 --workers 8 --batchSize 500

# Faster: more workers, larger batches
./testsuite --module datagen \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema bobo \
  --rowsNumber 2000000 --workers 16 --batchSize 1000

# Append without truncating existing data
./testsuite --module datagen \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema bobo \
  --rowsNumber 500000 --truncate=false
```

Create the schema first:

```bash
mysql -h 127.0.0.1 -u root -p < tools/data_test_schema.sql
```

| Flag | Default | Description |
|---|---|---|
| `--rowsNumber` | `10000` | Target number of users to generate |
| `--workers` | `8` | Parallel goroutines for INSERT batching |
| `--batchSize` | `500` | Rows per INSERT statement |
| `--truncate` | `true` | Truncate tables before loading |

---

## Output modes

**Human-readable** (default): formatted tables, live interval stats, progress bar (loop-based mode), histograms, and actionable warnings.

**CSV** (`--reportCSV`): no headers, no progress bar. Two row types share the same stdout stream:

```bash
# Collect everything
./testsuite --module connectionpool ... --reportCSV > results.csv

# Split by row type for analysis
grep '^interval,' results.csv > intervals.csv
grep '^summary,'  results.csv > summaries.csv

# Load in Python
import pandas as pd
df = pd.read_csv('summaries.csv', header=None)
# Column names are in the # comment lines at the top of results.csv
```

---

## Bugs and contributions

Please report issues and suggest new tests at the project repository.
