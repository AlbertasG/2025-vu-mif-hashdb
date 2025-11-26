package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/linxGnu/grocksdb"
	_ "github.com/mattn/go-sqlite3"
)

type FileData struct {
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	PackageID int64  `json:"package_id"`
}

func main() {
	// Open SQLite
	sqliteDB, err := sql.Open("sqlite3", "data/nist_rds_subset_50mb.db")
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteDB.Close()

	// Open RocksDB
	opts := grocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	// Initialize Bloom filter (10 bits per key is a good default)
	bbto := grocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(grocksdb.NewBloomFilter(10))
	bbto.SetCacheIndexAndFilterBlocks(true)
	opts.SetBlockBasedTableFactory(bbto)

	rocksDB, err := grocksdb.OpenDb(opts, "data/nist_rds_rocksdb")
	if err != nil {
		log.Fatal(err)
	}
	defer rocksDB.Close()

	wo := grocksdb.NewDefaultWriteOptions()

	// Query all files
	rows, err := sqliteDB.Query("SELECT sha256, sha1, md5, crc32, file_name, file_size, package_id FROM FILE")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var sha256, sha1, md5, crc32, fileName string
		var fileSize, packageID int64

		if err := rows.Scan(&sha256, &sha1, &md5, &crc32, &fileName, &fileSize, &packageID); err != nil {
			log.Fatal(err)
		}

		// Key = joined hashes
		key := sha256 + sha1 + md5 + crc32

		// Value = JSON of file data
		data := FileData{
			FileName:  fileName,
			FileSize:  fileSize,
			PackageID: packageID,
		}
		value, _ := json.Marshal(data)

		if err := rocksDB.Put(wo, []byte(key), value); err != nil {
			log.Fatal(err)
		}

		count++
		if count%10000 == 0 {
			fmt.Printf("Migrated %d records\n", count)
		}
	}

	fmt.Printf("Done! Migrated %d records total\n", count)
}
