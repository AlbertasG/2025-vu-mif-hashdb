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
	// Open SQLite
	sqliteDB, err := sql.Open("sqlite3", "data/nist_rds_subset.db")
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

	// Query all files with JOINs to get complete information
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
	`
	rows, err := sqliteDB.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var sha256, sha1, md5, crc32, fileName string
		var packageName, packageVersion, language, applicationType string
		var osName, osVersion, manufacturerName string
		var fileSize, packageID int64

		if err := rows.Scan(
			&sha256, &sha1, &md5, &crc32,
			&fileName, &fileSize, &packageID,
			&packageName, &packageVersion, &language, &applicationType,
			&osName, &osVersion, &manufacturerName,
		); err != nil {
			log.Fatal(err)
		}

		// Key = joined hashes
		key := sha256 + sha1 + md5 + crc32

		// Value = JSON of file data with all related information
		data := FileData{
			SHA256:           sha256,
			SHA1:             sha1,
			MD5:              md5,
			CRC32:            crc32,
			FileName:         fileName,
			FileSize:         fileSize,
			PackageID:        packageID,
			PackageName:      packageName,
			PackageVersion:   packageVersion,
			Language:         language,
			ApplicationType:  applicationType,
			OSName:           osName,
			OSVersion:        osVersion,
			ManufacturerName: manufacturerName,
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
