# Hash Database Performance Benchmark

This benchmark compares the performance of three different approaches for querying hash data:

1. **SQLite with JOINs** - Traditional relational database with multi-table joins
2. **RocksDB with Bloom filter** - Key-value store with optimized bloom filter
3. **RocksDB without Bloom filter** - Key-value store without bloom filter

## Prerequisites

Before running the benchmark, ensure you have:

1. RocksDB installed (via Homebrew on macOS):
   ```bash
   brew install rocksdb
   ```

2. CGO environment variables set (for RocksDB):
   ```bash
   source setup_env.sh
   ```
   Or manually:
   ```bash
   export CGO_CFLAGS="-I/opt/homebrew/include"
   export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
   ```
   Note: The `run_benchmark.sh` script sets these automatically.

3. Migrated data to RocksDB:
   ```bash
   go run migrate_to_rocksdb.go
   ```

4. SQLite database at `data/nist_rds_subset.db`

## Running the Benchmark

### Quick Start

```bash
./run_benchmark.sh
```

This will run 1000 queries by default (70% existing hashes, 30% non-existing).

### Custom Number of Queries

```bash
./run_benchmark.sh 5000    # Run 5000 queries
./run_benchmark.sh 100     # Run 100 queries
```

### Direct Go Execution

If running directly with Go (not using the shell script), set the CGO variables first:

```bash
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"

go run benchmark.go        # 1000 queries (default)
go run benchmark.go 5000   # 5000 queries
```

## What the Benchmark Tests

The benchmark generates a realistic workload:
- **70% existing hashes** - These will be found in the database
- **30% non-existing hashes** - These won't be found (tests negative lookups)

For each database system, it measures:
- Total execution time
- Average query time
- Min/Max query times
- Queries per second
- Hit rate (found vs not found)

## Understanding the Results

### Expected Results

**RocksDB with Bloom Filter** should be the fastest because:
- Key-value lookups are O(1) vs SQLite's B-tree O(log n)
- Bloom filter quickly rejects non-existing keys
- No JOIN overhead
- Data is denormalized (stored as JSON)

**RocksDB without Bloom Filter** should be slower for negative lookups:
- Must perform full disk I/O for non-existing keys
- Bloom filter advantage becomes clear here
- Note: Uses the same database, just opens it without bloom filter policy

**SQLite with JOINs** should be slower because:
- B-tree index lookups
- Multiple table JOINs (FILE → PKG → OS → MFG)
- Row assembly overhead

### Sample Output

```
1. SQLite (with JOINs)
   ────────────────────────────────────────────────────────────────────
   Total Queries:    1000
   Found:            700 (70.0%)
   Total Time:       2.5s
   Average Time:     2.5ms
   Queries/sec:      400.00

2. RocksDB with Bloom filter
   ────────────────────────────────────────────────────────────────────
   Total Queries:    1000
   Found:            700 (70.0%)
   Total Time:       150ms
   Average Time:     150µs
   Queries/sec:      6666.67

COMPARISON
   RocksDB with Bloom filter vs SQLite (with JOINs):
   Speedup: 16.67x 🚀
```

## Files

- `benchmark.go` - Main benchmark implementation
- `run_benchmark.sh` - Convenience script to run benchmarks
- `BENCHMARK_README.md` - This file

## Notes

- Results may vary based on:
  - Database size
  - System hardware (CPU, RAM, SSD vs HDD)
  - OS and system load
  - Go compiler optimizations
- For production use cases, consider warming up caches before benchmarking

## Advanced Usage

To test specific scenarios, modify `benchmark.go`:
- Adjust the hit rate ratio (currently 70/30)
- Test only existing or only non-existing hashes
- Add more database configurations
- Test with different Bloom filter parameters

