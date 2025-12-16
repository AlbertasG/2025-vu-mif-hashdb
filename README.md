# HashDB - NIST RDS Hash Lookup Tool

A high-performance hash lookup tool using RocksDB with Bloom filters for fast file identification against the NIST Reference Data Set (RDS).

## Cloning without data sample

cloning the repository with also download 1GB+++ dataset, if you want to skip and only download source code / documentation / latex report use the following command
`GIT_LFS_SKIP_SMUDGE=1 git clone git@github.com:AlbertasG/2025-vu-mif-hashdb.git`

## Overview


This tool provides two main components:

1. **`migrate_to_rocksdb.go`** - Migrates data from SQLite to RocksDB
2. **`hashdb.go`** - Command-line tool for hash lookups

The system uses a composite key of all four hashes (SHA256 + SHA1 + MD5 + CRC32) for exact matching, with Bloom filters for fast negative lookups.

## Prerequisites

### 1. Install RocksDB

**macOS (Homebrew):**
```bash
brew install rocksdb
```

**Ubuntu/Debian:**
```bash
sudo apt-get install librocksdb-dev
```

### 2. Set CGO Environment Variables

RocksDB requires CGO. Set the environment variables:

```bash
source setup_env.sh
```

Or manually (macOS with Homebrew):
```bash
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib -lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy -llz4 -lzstd"
```

### 3. Source Data

The TEST dataset used as subset from NIST RDS v3 database in this repository is stored with git-lfs, so you have to install git lfs extension on your machine if you want to pull the data via git. 

Ensure you have the SQLite database at `data/nist_rds_subset.db` containing the NIST RDS data with the following tables:
- `FILE` - File hashes and metadata
- `PKG` - Package information
- `OS` - Operating system details
- `MFG` - Manufacturer information

---

## Step 1: Migrate Data to RocksDB

The migration tool reads from SQLite and creates an optimized RocksDB database with Bloom filters.

### Run Migration

```bash
source setup_env.sh
go run migrate_to_rocksdb.go
```

### What It Does

1. Opens the SQLite database (`data/nist_rds_subset.db`)
2. Creates a new RocksDB database (`data/nist_rds_rocksdb/`)
3. Joins all related tables (FILE → PKG → OS → MFG)
4. Stores each record with:
   - **Key**: Concatenated hashes (`SHA256 + SHA1 + MD5 + CRC32`)
   - **Value**: JSON containing all file metadata

### Progress Output

```
Migrated 10000 records
Migrated 20000 records
...
Done! Migrated 1234567 records total
```

---

## Step 2: Use HashDB for Lookups

### Build the Tool (Optional)

```bash
source setup_env.sh
go build -o hashdb hashdb.go
```

### Single Hash Lookup

Look up a file by providing all four hashes:

```bash
./hashdb <sha256> <sha1> <md5> <crc32>
```

**Example:**
```bash
./hashdb \
  0000202989C986A8DD5E370FF72AF50A939E1B401ADFA43088CE4573C4303EB8 \
  DA39A3EE5E6B4B0D3255BFEF95601890AFD80709 \
  D41D8CD98F00B204E9800998ECF8427E \
  00000000
```

Or run directly with Go:
```bash
go run hashdb.go <sha256> <sha1> <md5> <crc32>
```

### Bulk Lookup from File

Process multiple hashes from a file:

```bash
./hashdb -f <hashfile>
```

**Example:**
```bash
./hashdb -f hashes_to_check.txt
```

### Input File Format

Create a text file with comma-separated hashes (one lookup per line):

```
# Lines starting with # are comments
# Format: SHA256,SHA1,MD5,CRC32

0000202989C986A8DD5E370FF72AF50A939E1B401ADFA43088CE4573C4303EB8,DA39A3EE5E6B4B0D3255BFEF95601890AFD80709,D41D8CD98F00B204E9800998ECF8427E,00000000
00002A58A9E14B95F1372A4BDBB041911497BA71771EF840226C336956635BE4,A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2,1234567890ABCDEF1234567890ABCDEF,DEADBEEF
```

### Output Examples

**Found:**
```
FOUND 0000202989C986A8DD5E370FF72AF50A939E1B401ADFA43088CE4573C4303EB8
  File: example.dll (12345 bytes)
  SHA256: 0000202989C986A8DD5E370FF72AF50A939E1B401ADFA43088CE4573C4303EB8
  SHA1:   DA39A3EE5E6B4B0D3255BFEF95601890AFD80709
  MD5:    D41D8CD98F00B204E9800998ECF8427E
  CRC32:  00000000
  Package: Windows Update 10.0.19041 (ID: 100)
  Language: English
  Type: Operating System
  OS: Windows 10
  Manufacturer: Microsoft Corporation
```

**Not Found:**
```
NOT FOUND DEADBEEF12345678901234567890123456789012345678901234567890123456
```
