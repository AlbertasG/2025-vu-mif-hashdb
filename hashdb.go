// Usage:
//   hashdb <sha256> <sha1> <md5> <crc32>   - Exact lookup (uses bloom filter)
//   hashdb -f <file>                        - Bulk lookup from file
//
// File format for bulk lookup:
//   # Comment lines start with #
//   SHA256,SHA1,MD5,CRC32                   - Comma-separated hashes per line

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/linxGnu/grocksdb"
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

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Open RocksDB
	opts := grocksdb.NewDefaultOptions()
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(grocksdb.NewBloomFilter(10))
	opts.SetBlockBasedTableFactory(bbto)

	db, err := grocksdb.OpenDbForReadOnly(opts, "data/nist_rds_rocksdb", false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ro := grocksdb.NewDefaultReadOptions()

	// Bulk lookup from file
	if os.Args[1] == "-f" && len(os.Args) >= 3 {
		file, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			
			parts := strings.Split(line, ",")
			if len(parts) == 4 {
				lookup(db, ro, 
					strings.ToUpper(strings.TrimSpace(parts[0])),
					strings.ToUpper(strings.TrimSpace(parts[1])),
					strings.ToUpper(strings.TrimSpace(parts[2])),
					strings.ToUpper(strings.TrimSpace(parts[3])))
			} else {
				fmt.Fprintf(os.Stderr, "Invalid line (need SHA256,SHA1,MD5,CRC32): %s\n", line)
			}
		}
		return
	}

	// Exact lookup with all 4 hashes
	if len(os.Args) == 5 {
		sha256 := strings.ToUpper(strings.TrimSpace(os.Args[1]))
		sha1 := strings.ToUpper(strings.TrimSpace(os.Args[2]))
		md5 := strings.ToUpper(strings.TrimSpace(os.Args[3]))
		crc32 := strings.ToUpper(strings.TrimSpace(os.Args[4]))
		lookup(db, ro, sha256, sha1, md5, crc32)
		return
	}

	printUsage()
	os.Exit(1)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  hashdb <sha256> <sha1> <md5> <crc32>   - Exact lookup")
	fmt.Println("  hashdb -f <file>                       - Bulk lookup from file")
	fmt.Println()
	fmt.Println("File format (comma-separated):")
	fmt.Println("  SHA256,SHA1,MD5,CRC32")
}

func lookup(db *grocksdb.DB, ro *grocksdb.ReadOptions, sha256, sha1, md5, crc32 string) {
	key := sha256 + sha1 + md5 + crc32
	
	value, err := db.Get(ro, []byte(key))
	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}
	defer value.Free()

	if !value.Exists() {
		fmt.Printf("NOT FOUND %s\n", sha256)
		return
	}

	var data FileData
	json.Unmarshal(value.Data(), &data)
	
	fmt.Printf("FOUND %s\n", sha256)
	fmt.Printf("  File: %s (%d bytes)\n", data.FileName, data.FileSize)
	fmt.Printf("  SHA256: %s\n", data.SHA256)
	fmt.Printf("  SHA1:   %s\n", data.SHA1)
	fmt.Printf("  MD5:    %s\n", data.MD5)
	fmt.Printf("  CRC32:  %s\n", data.CRC32)
	if data.PackageName != "" {
		fmt.Printf("  Package: %s %s (ID: %d)\n", data.PackageName, data.PackageVersion, data.PackageID)
	} else {
		fmt.Printf("  Package ID: %d\n", data.PackageID)
	}
	if data.Language != "" {
		fmt.Printf("  Language: %s\n", data.Language)
	}
	if data.ApplicationType != "" {
		fmt.Printf("  Type: %s\n", data.ApplicationType)
	}
	if data.OSName != "" {
		fmt.Printf("  OS: %s %s\n", data.OSName, data.OSVersion)
	}
	if data.ManufacturerName != "" {
		fmt.Printf("  Manufacturer: %s\n", data.ManufacturerName)
	}
}
