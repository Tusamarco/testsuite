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

The binary requires **Go 1.21+** (matching the `go-sql-driver/mysql` v1.9 dependency).

---

## Common flags

These flags apply to every module.

| Flag | Default | Description |
|---|---|---|
| `--url` | | MySQL address as `host:port` |
| `--user` | `app_test` | Database user |
| `--password` | `test` | Database password |
| `--schema` | `mysql` | Default schema |
| `--loops` | `50` | Iterations per worker |
| `--sleep` | `0` | Milliseconds to sleep between iterations |
| `--verbose` | `false` | Extended per-iteration output |
| `--summary` | `false` | Print summary statistics at the end |
| `--reportCSV` | `false` | Machine-readable CSV output (suppresses progress lines) |

---

## Connection Pool module

### What it measures

Every operation is broken into three timed phases:

| Phase | What it captures |
|---|---|
| **Acquire** | `db.Conn()` — time to obtain a connection from the `sql.DB` pool. Includes queue wait when all connections are in use. |
| **Query** | Round-trip for the SQL payload. Isolates network + server execution time from pool overhead. |
| **Release** | `conn.Close()` — returns the connection to the pool. |

For each phase the report shows **Avg, P50, P95, P99, Max** latency in nanoseconds.

After each scenario the module queries MySQL status variables and `sql.DB.Stats()` to surface:

- `Threads_connected`, `Threads_running`, `Threads_cached`, `Threads_created`
- `thread_cache_size` and utilisation percentage
- MySQL thread-cache hit rate = `(ΔConnections − ΔThreads_created) / ΔConnections × 100`
- App pool: `OpenConns`, `InUse`, `Idle`, `WaitCount`, `WaitDuration`
- Automatic warnings when the app pool is too small or the MySQL thread cache hit rate drops below 80 %

### Module-specific flags

| Flag | Default | Description |
|---|---|---|
| `--workers` | `8` | Concurrent goroutines hitting the database simultaneously |
| `--maxOpenConns` | `10` | `sql.DB` `SetMaxOpenConns` — hard cap on open connections |
| `--maxIdleConns` | `5` | `sql.DB` `SetMaxIdleConns` — connections kept open between operations |
| `--connMaxLifetimeSec` | `0` | `sql.DB` `SetConnMaxLifetime` in seconds (`0` = unlimited) |
| `--payloadSize` | `small` | Query data volume: `small` (~10 B), `medium` (~1 KB), `large` (~64 KB), `xlarge` (~1 MB) |
| `--rampWorkers` | `false` | Run multiple scenarios at increasing concurrency levels up to `--workers` |
| `--warmupLoops` | `5` | Iterations run before measurement to pre-heat the pool |

---

### Scenarios

#### Baseline: serial connections, small payload

Understand raw connection overhead with a single goroutine.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 1 --loops 200 --payloadSize small \
  --maxOpenConns 5 --maxIdleConns 2
```

Use this as your baseline. Acquire time here reflects pure pool + network round-trip with no contention.

---

#### Pool pressure: workers exceed pool size

Set `--workers` larger than `--maxOpenConns` to force goroutines to queue for connections. This makes `WaitCount` and `WaitDuration` rise visibly and P99 acquire time spike.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 100 --payloadSize small \
  --maxOpenConns 10 --maxIdleConns 5
```

Expected output (approximate):

```
  [!] App pool contention: 18 goroutines waited (total 423ms).
      Consider increasing --maxOpenConns (currently 10).
```

Increase `--maxOpenConns` to match `--workers` and observe the wait disappear:

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 100 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 16
```

---

#### Concurrency ramp: find the saturation point

`--rampWorkers` runs the same scenario at 1 → 4 → 16 → 32 → 64 → … workers up to `--workers`. Compare throughput (QPS) and P99 acquire time at each level to find where the server or pool saturates.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 128 --rampWorkers \
  --loops 200 --payloadSize small \
  --maxOpenConns 128 --maxIdleConns 64
```

Pipe to CSV to graph the results:

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 128 --rampWorkers --loops 200 \
  --maxOpenConns 128 --maxIdleConns 64 \
  --reportCSV > ramp.csv
```

CSV columns: `workers, loops, payload_size, max_open_conns, max_idle_conns, avg_acquire_ns, p50_acquire_ns, p95_acquire_ns, p99_acquire_ns, max_acquire_ns, avg_query_ns, p50_query_ns, p95_query_ns, p99_query_ns, max_query_ns, avg_total_ns, throughput_qps, bytes_per_sec, app_open, app_in_use, app_idle, app_wait_count, app_wait_ms, mysql_connected_before, mysql_cached_before, mysql_created_before, mysql_connected_after, mysql_cached_after, mysql_created_after, new_threads_created, thread_cache_hit_pct, errors`

---

#### Data volume: payload size effect on latency

Fix concurrency and vary the payload to understand when data transfer dominates over connection overhead.

```bash
# Small (~10 B) — connection overhead visible
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 200 --payloadSize small \
  --maxOpenConns 16 --maxIdleConns 8

# Medium (~1 KB)
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 200 --payloadSize medium \
  --maxOpenConns 16 --maxIdleConns 8

# Large (~64 KB) — query time dominates
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 200 --payloadSize large \
  --maxOpenConns 16 --maxIdleConns 8

# XLarge (~1 MB)
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 200 --payloadSize xlarge \
  --maxOpenConns 16 --maxIdleConns 8
```

---

#### Connection lifetime: effect of `ConnMaxLifetime`

When `--connMaxLifetimeSec` is set, `sql.DB` closes and recreates connections periodically. This forces MySQL to spawn new threads — visible as `MaxLifetimeClosed` increasing and the thread-cache hit rate dropping.

```bash
# Unlimited lifetime (default) — pool reuses connections aggressively
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 500 --payloadSize small \
  --maxOpenConns 16 --maxIdleConns 8 \
  --connMaxLifetimeSec 0

# 5-second lifetime — forces periodic reconnects
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 16 --loops 500 --payloadSize small \
  --maxOpenConns 16 --maxIdleConns 8 \
  --connMaxLifetimeSec 5
```

Watch `new_threads_created` and `thread_cache_hit_pct` differ between the two runs. A low hit rate with `--connMaxLifetimeSec 5` but a high one without it confirms the pool is forcing reconnects faster than the thread cache can absorb.

---

#### MySQL thread cache: diagnosing a small `thread_cache_size`

On a MySQL server with `thread_cache_size=8`, run 32 concurrent workers. The thread cache fills up, MySQL starts spawning new OS threads for each new connection, and `Threads_created` grows rapidly.

```bash
./testsuite --module connectionpool \
  --url 127.0.0.1:3306 --user root --password secret \
  --workers 32 --loops 300 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 0
```

Setting `--maxIdleConns 0` forces `sql.DB` to close connections after every operation, so every new `db.Conn()` creates a fresh MySQL connection. This maximises pressure on the thread cache and makes the difference between `thread_cache_size=8` and `thread_cache_size=64` clearly visible in `Threads_created`.

Expected warning when hit rate is low:

```
  [!] Low MySQL thread cache hit rate (34.2%). Consider increasing thread_cache_size=8.
```

---

#### ProxySQL / connection router

Point `--url` at a proxy rather than MySQL directly. The acquire time now includes proxy routing overhead. Compare with a direct connection to quantify proxy cost:

```bash
# Direct to MySQL
./testsuite --module connectionpool \
  --url mysql-primary:3306 --user root --password secret \
  --workers 32 --loops 200 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 16 --reportCSV > direct.csv

# Via ProxySQL
./testsuite --module connectionpool \
  --url proxysql-host:6033 --user root --password secret \
  --workers 32 --loops 200 --payloadSize small \
  --maxOpenConns 32 --maxIdleConns 16 --reportCSV > proxy.csv
```

Compare `avg_acquire_ns` and `p99_acquire_ns` between the two files.

---

### Reading the output

```
══════════════════════════════════════════════════════════════════════
  Workers: 32   | Loops: 200    | Payload: small    | Duration: 1.243s
  App Pool: MaxOpen=32   MaxIdle=16   Operations: 6400   Errors: 0
  Throughput: 5151.5 QPS | 71.2 KB/s
──────────────────────────────────────────────────────────────────────
  Phase          Avg (ns)     P50 (ns)     P95 (ns)     P99 (ns)     Max (ns)
  Acquire           41823        28104       112847       287531      1842033
  Query            139204       127340       221890       389122      2103441
  Round-trip       184209       162881       348291       701244      3981204
──────────────────────────────────────────────────────────────────────
  App Pool (sql.DB) — end-of-scenario snapshot:
    OpenConns=32    InUse=0     Idle=16
    WaitCount(delta)=0        WaitDuration(delta)=0s
    MaxIdleClosed(delta)=0    MaxLifetimeClosed(delta)=0
──────────────────────────────────────────────────────────────────────
  MySQL Server (thread_cache_size=32):
    Before: connected=3    running=1    cached=8     created=112
    After:  connected=3    running=1    cached=16    created=112
    New connections to MySQL: 6400   New threads spawned: 0    Cache hit rate: 100.0%
══════════════════════════════════════════════════════════════════════
```

Key things to look at:

- **Acquire P99 vs Avg**: a large gap means occasional pool stalls; increase `--maxOpenConns`.
- **WaitCount > 0**: goroutines had to queue — the pool is the bottleneck.
- **New threads spawned > 0**: MySQL had to create OS threads; the thread cache is too small or connections are closing too frequently.
- **Cache hit rate < 80 %**: increase `thread_cache_size` on the MySQL server.
- **Query P99 >> P50**: server-side query execution is inconsistent (lock contention, slow disk, etc.).

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
| `--sleep` | Milliseconds to wait after table load before testing |
| `--rowsNumber` | Rows pre-loaded into the test table (default 10 000) |

---

## DataGen module

Seeds reference tables (continents, countries, cities, streets, names) then generates addresses and users in parallel. All data uses real European cities and names.

```bash
# Default: 2 M users, 500 K addresses, 8 workers
./testsuite --module datagen \
  --url 127.0.0.1:3306 --user root --password secret \
  --schema bobo \
  --rowsNumber 2000000 --workers 8 --batchSize 500

# Faster load with more workers and larger batches
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

The schema to create before running is in `tools/data_test_schema.sql`.

---

## Output modes

All modules support two output modes controlled by `--reportCSV`.

**Human-readable** (default): formatted tables, progress lines, and interpretation hints.

**CSV** (`--reportCSV`): one header row followed by one data row per scenario. Progress lines are suppressed. Redirect to a file and import into a spreadsheet or `gnuplot` for graphing.

```bash
./testsuite --module connectionpool ... --reportCSV > results.csv
```

---

## Bugs and contributions

Please report issues and suggest new tests at the project repository.
