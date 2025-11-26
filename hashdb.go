// To run: go run hashdb.go -f test_hashes.txt

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
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	PackageID int64  `json:"package_id"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: hashdb <hash> or hashdb -f <file>")
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
			hash := strings.ToUpper(strings.TrimSpace(scanner.Text()))
			if hash != "" && !strings.HasPrefix(hash, "#") {
				lookup(db, ro, hash)
			}
		}
		return
	}

	// Single hash lookup
	hash := strings.ToUpper(strings.TrimSpace(os.Args[1]))
	lookup(db, ro, hash)
}

func lookup(db *grocksdb.DB, ro *grocksdb.ReadOptions, hash string) {
	// SHA256 prefix lookup
	iter := db.NewIterator(ro)
	defer iter.Close()

	iter.Seek([]byte(hash))
	if iter.Valid() {
		key := string(iter.Key().Data())
		if strings.HasPrefix(key, hash) {
			var data FileData
			json.Unmarshal(iter.Value().Data(), &data)
			fmt.Printf("FOUND %s -> %s (%d bytes)\n", hash, data.FileName, data.FileSize)
			return
		}
	}

	fmt.Printf("NOT FOUND %s\n", hash)
}
