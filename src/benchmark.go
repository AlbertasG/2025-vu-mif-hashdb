package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

type FileData struct {
	SHA256           string `json:"sha256"`
	SHA1             string `json:"sha1"`
	MD5              string `json:"md5"`
	CRC32            string `json:"crc32"`
	FileName         string `json:"file_name"`
	FileSize         int64  `json:"file_size"`
	PackageID        int64  `json:"package_id"`
	PackageName      string `json:"package_name"`
	PackageVersion   string `json:"package_version"`
	Language         string `json:"language"`
	ApplicationType  string `json:"application_type"`
	OSName           string `json:"os_name"`
	OSVersion        string `json:"os_version"`
	ManufacturerName string `json:"manufacturer_name"`
}

type BenchmarkResult struct {
	Name         string
	TotalQueries int
	FoundCount   int
	TotalTime    time.Duration
	AvgTime      time.Duration
	MinTime      time.Duration
	MaxTime      time.Duration
}

var useDirectIO bool

func printRocksDBMemoryStats() {
	fmt.Println("=== RocksDB Memory Statistics ===")
	
	// Open DB with bloom filter to get stats
	opts := grocksdb.NewDefaultOptions()
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(grocksdb.NewBloomFilter(10))
	bbto.SetCacheIndexAndFilterBlocks(true)
	opts.SetBlockBasedTableFactory(bbto)

	db, err := grocksdb.OpenDbForReadOnly(opts, "../data/nist_rds_rocksdb", false)
	if err != nil {
		fmt.Printf("Could not open DB for stats: %v\n\n", err)
		return
	}
	defer db.Close()

	// Get key count first
	numKeysStr := db.GetProperty("rocksdb.estimate-num-keys")
	var numKeys int64
	fmt.Sscanf(numKeysStr, "%d", &numKeys)

	// Get memory properties
	fmt.Println("  Database info:")
	fmt.Printf("    Estimated keys:              %d\n", numKeys)
	
	sstSize := db.GetProperty("rocksdb.total-sst-files-size")
	var sstSizeNum int64
	fmt.Sscanf(sstSize, "%d", &sstSizeNum)
	fmt.Printf("    Total SST files on disk:     %s\n", formatBytes(sstSizeNum))

	// Calculate bloom filter size
	if numKeys > 0 {
		bitsPerKey := 10 // We configured 10 bits per key
		theoreticalBloomSize := (numKeys * int64(bitsPerKey)) / 8
		
		fmt.Println("\n  Bloom filter memory:")
		fmt.Printf("    Keys:           %d\n", numKeys)
		fmt.Printf("    Bits per key:   %d\n", bitsPerKey)
		fmt.Printf("    Filter size:    %s  (formula: keys × bits / 8)\n", formatBytes(theoreticalBloomSize))
		
		// Show percentage of SST file size
		percentage := float64(theoreticalBloomSize) / float64(sstSizeNum) * 100
		fmt.Printf("    %% of DB size:   %.2f%%\n", percentage)
	}

	fmt.Println()
}

func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

func main() {
	numQueries := 1000
	
	// Parse arguments
	for i, arg := range os.Args[1:] {
		if arg == "--direct-io" {
			useDirectIO = true
		} else if i == 0 || (i == 1 && !useDirectIO) {
			fmt.Sscanf(arg, "%d", &numQueries)
		}
	}

	fmt.Printf("=== Hash Database Performance Benchmark ===\n")
	fmt.Printf("Number of queries: %d\n", numQueries)
	if useDirectIO {
		fmt.Printf("Direct I/O: ENABLED (bypasses OS cache)\n")
	} else {
		fmt.Printf("Direct I/O: disabled (using OS cache)\n")
	}
	fmt.Println()

	// Generate test hashes
	testHashes := generateTestHashes(numQueries)
	fmt.Printf("Generated %d test hashes (mix of existing and non-existing)\n\n", len(testHashes))

	// Print RocksDB memory statistics
	printRocksDBMemoryStats()

	// Run benchmarks
	results := []BenchmarkResult{}

	fmt.Println("Running SQLite benchmark...")
	results = append(results, benchmarkSQLite(testHashes))

	fmt.Println("Running RocksDB Get WITH Bloom filter...")
	results = append(results, benchmarkRocksDBGet(testHashes, true))

	fmt.Println("Running RocksDB Get WITHOUT Bloom filter...")
	results = append(results, benchmarkRocksDBGet(testHashes, false))

	// Print results
	printResults(results)
}

type TestHash struct {
	SHA256  string
	SHA1    string
	MD5     string
	CRC32   string
	FullKey string
	Exists  bool
}

func generateTestHashes(count int) []TestHash {
	sqliteDB, err := sql.Open("sqlite3", "../data/nist_rds_subset.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteDB.Close()

	// Get random existing hashes (70% of queries)
	existingCount := int(float64(count) * 0.7)
	rows, err := sqliteDB.Query("SELECT sha256, sha1, md5, crc32 FROM FILE ORDER BY RANDOM() LIMIT ?", existingCount)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	hashes := []TestHash{}
	for rows.Next() {
		var sha256, sha1, md5, crc32 string
		rows.Scan(&sha256, &sha1, &md5, &crc32)
		hashes = append(hashes, TestHash{
			SHA256:  sha256,
			SHA1:    sha1,
			MD5:     md5,
			CRC32:   crc32,
			FullKey: sha256 + sha1 + md5 + crc32,
			Exists:  true,
		})
	}

	// Generate random non-existing hashes (30% of queries)
	nonExistingCount := count - len(hashes)
	for i := 0; i < nonExistingCount; i++ {
		sha256 := generateRandomHash(64)
		sha1 := generateRandomHash(40)
		md5 := generateRandomHash(32)
		crc32 := generateRandomHash(8)
		hashes = append(hashes, TestHash{
			SHA256:  sha256,
			SHA1:    sha1,
			MD5:     md5,
			CRC32:   crc32,
			FullKey: sha256 + sha1 + md5 + crc32,
			Exists:  false,
		})
	}

	// Shuffle the hashes
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(hashes), func(i, j int) {
		hashes[i], hashes[j] = hashes[j], hashes[i]
	})

	return hashes
}

func generateRandomHash(length int) string {
	const chars = "0123456789ABCDEF"
	hash := make([]byte, length)
	for i := range hash {
		hash[i] = chars[rand.Intn(len(chars))]
	}
	return string(hash)
}

func benchmarkSQLite(testHashes []TestHash) BenchmarkResult {
	sqliteDB, err := sql.Open("sqlite3", "../data/nist_rds_subset.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteDB.Close()

	query := `
		SELECT 
			f.sha256, f.sha1, f.md5, f.crc32, 
			f.file_name, f.file_size, f.package_id,
			p.name as package_name, p.version as package_version, 
			p.language, p.application_type,
			o.name as os_name, o.version as os_version,
			m.name as manufacturer_name
		FROM FILE f
		LEFT JOIN PKG p ON f.package_id = p.package_id
		LEFT JOIN OS o ON p.operating_system_id = o.operating_system_id 
			AND p.manufacturer_id = o.manufacturer_id
		LEFT JOIN MFG m ON o.manufacturer_id = m.manufacturer_id
		WHERE f.sha256 = ? AND f.sha1 = ? AND f.md5 = ? AND f.crc32 = ?
	`

	stmt, err := sqliteDB.Prepare(query)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	result := BenchmarkResult{
		Name:         "SQLite (exact match)",
		TotalQueries: len(testHashes),
		MinTime:      time.Hour,
		MaxTime:      0,
	}

	start := time.Now()
	for _, th := range testHashes {
		queryStart := time.Now()
		rows, err := stmt.Query(th.SHA256, th.SHA1, th.MD5, th.CRC32)
		if err != nil {
			log.Fatal(err)
		}

		found := false
		if rows.Next() {
			found = true
			var sha256, sha1, md5, crc32, fileName string
			var packageName, packageVersion, language, applicationType string
			var osName, osVersion, manufacturerName string
			var fileSize, packageID int64

			rows.Scan(
				&sha256, &sha1, &md5, &crc32,
				&fileName, &fileSize, &packageID,
				&packageName, &packageVersion, &language, &applicationType,
				&osName, &osVersion, &manufacturerName,
			)
		}
		rows.Close()

		queryTime := time.Since(queryStart)
		if found {
			result.FoundCount++
		}
		if queryTime < result.MinTime {
			result.MinTime = queryTime
		}
		if queryTime > result.MaxTime {
			result.MaxTime = queryTime
		}
	}
	result.TotalTime = time.Since(start)
	result.AvgTime = result.TotalTime / time.Duration(result.TotalQueries)

	return result
}

func benchmarkRocksDBGet(testHashes []TestHash, useBloomFilter bool) BenchmarkResult {
	opts := grocksdb.NewDefaultOptions()
	
	if useDirectIO {
		opts.SetUseDirectReads(true)
	}
	
	name := "RocksDB WITH Bloom filter"
	
	if useBloomFilter {
		bbto := grocksdb.NewDefaultBlockBasedTableOptions()
		bbto.SetFilterPolicy(grocksdb.NewBloomFilter(10))
		bbto.SetCacheIndexAndFilterBlocks(true)
		opts.SetBlockBasedTableFactory(bbto)
	} else {
		name = "RocksDB WITHOUT Bloom filter"
	}

	db, err := grocksdb.OpenDbForReadOnly(opts, "../data/nist_rds_rocksdb", false)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ro := grocksdb.NewDefaultReadOptions()

	result := BenchmarkResult{
		Name:         name,
		TotalQueries: len(testHashes),
		MinTime:      time.Hour,
		MaxTime:      0,
	}

	start := time.Now()
	for _, th := range testHashes {
		queryStart := time.Now()
		
		value, err := db.Get(ro, []byte(th.FullKey))
		
		found := false
		if err == nil && value.Exists() {
			found = true
			var data FileData
			json.Unmarshal(value.Data(), &data)
		}
		if value != nil {
			value.Free()
		}

		queryTime := time.Since(queryStart)
		if found {
			result.FoundCount++
		}
		if queryTime < result.MinTime {
			result.MinTime = queryTime
		}
		if queryTime > result.MaxTime {
			result.MaxTime = queryTime
		}
	}
	result.TotalTime = time.Since(start)
	result.AvgTime = result.TotalTime / time.Duration(result.TotalQueries)

	return result
}

func printResults(results []BenchmarkResult) {
	fmt.Println("\n╔═══════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                        BENCHMARK RESULTS                                  ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")

	for i, r := range results {
		fmt.Printf("\n%d. %s\n", i+1, r.Name)
		fmt.Println("   " + strings.Repeat("─", 70))
		fmt.Printf("   Total Queries:    %d\n", r.TotalQueries)
		fmt.Printf("   Found:            %d (%.1f%%)\n", r.FoundCount, float64(r.FoundCount)/float64(r.TotalQueries)*100)
		fmt.Printf("   Total Time:       %v\n", r.TotalTime)
		fmt.Printf("   Average Time:     %v\n", r.AvgTime)
		fmt.Printf("   Min Time:         %v\n", r.MinTime)
		fmt.Printf("   Max Time:         %v\n", r.MaxTime)
		fmt.Printf("   Queries/sec:      %.2f\n", float64(r.TotalQueries)/r.TotalTime.Seconds())
	}

	// Comparison
	if len(results) > 1 {
		fmt.Println("\n╔═══════════════════════════════════════════════════════════════════════════╗")
		fmt.Println("║                          COMPARISON                                       ║")
		fmt.Println("╚═══════════════════════════════════════════════════════════════════════════╝")

		baseline := results[0]
		for i := 1; i < len(results); i++ {
			speedup := float64(baseline.TotalTime) / float64(results[i].TotalTime)
			fmt.Printf("\n%s vs %s:\n", results[i].Name, baseline.Name)
			fmt.Printf("   Speedup: %.2fx %s\n", speedup, getSpeedupEmoji(speedup))
			fmt.Printf("   Time difference: %v\n", baseline.TotalTime-results[i].TotalTime)
		}
	}

	fmt.Println()
}

func getSpeedupEmoji(speedup float64) string {
	if speedup > 2 {
		return "🚀"
	} else if speedup > 1.5 {
		return "⚡"
	} else if speedup > 1.1 {
		return "✓"
	} else if speedup > 0.9 {
		return "≈"
	}
	return "🐌"
}
