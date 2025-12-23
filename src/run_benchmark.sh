#!/bin/bash

# Hash Database Performance Benchmark Runner
# Usage: ./run_benchmark.sh [number_of_queries] [--direct-io]
#
# Options:
#   --direct-io   Bypass OS cache to simulate real disk I/O
#                 (shows true Bloom filter benefit)

set -e

# Set CGO flags for RocksDB
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"

NUM_QUERIES=${1:-1000}
DIRECT_IO=""

# Check for --direct-io flag
for arg in "$@"; do
    if [ "$arg" = "--direct-io" ]; then
        DIRECT_IO="--direct-io"
    fi
done

echo "╔════════════════════════════════════════════════════════╗"
echo "║   Hash Database Performance Benchmark                  ║"
echo "╚════════════════════════════════════════════════════════╝"
echo ""

# Check if RocksDB exists
if [ ! -d "../data/nist_rds_rocksdb" ]; then
    echo "❌ Error: RocksDB not found at ../data/nist_rds_rocksdb"
    echo "Please run the migration first:"
    echo "  go run migrate_to_rocksdb.go"
    exit 1
fi

# Check if SQLite exists
if [ ! -f "../data/nist_rds_subset.db" ]; then
    echo "❌ Error: SQLite database not found at ../data/nist_rds_subset.db"
    exit 1
fi

echo "✓ Found databases"
echo ""
echo "Starting benchmark with $NUM_QUERIES queries..."
if [ -n "$DIRECT_IO" ]; then
    echo "Direct I/O: ENABLED (bypasses OS cache)"
else
    echo "Direct I/O: disabled (add --direct-io to enable)"
fi
echo ""
echo "This will test:"
echo "  1. SQLite (exact match)"
echo "  2. RocksDB WITH Bloom filter"
echo "  3. RocksDB WITHOUT Bloom filter"
echo ""
echo "Please wait..."
echo ""

# Run the benchmark
go run benchmark.go "$NUM_QUERIES" $DIRECT_IO

echo ""
echo "Benchmark complete!"

