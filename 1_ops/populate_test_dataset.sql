sqlite3 nist_rds_subset.db < RDS_2025.03.1_modern_minimal.schema.sql
sqlite3 nist_rds_subset.db
ATTACH DATABASE 'RDS_2025.03.1_modern_minimal.db' AS old_db;
INSERT INTO VERSION SELECT * FROM old_db.VERSION;
INSERT INTO MFG SELECT * FROM old_db.MFG;
INSERT INTO OS SELECT * FROM old_db.OS;
INSERT INTO PKG SELECT * FROM old_db.PKG WHERE package_id IN ( 289308, 302124, 288596 );
INSERT INTO FILE (sha256, sha1, md5, crc32, file_name, file_size, package_id)
    SELECT 
        old_db.FILE.sha256, 
        old_db.FILE.sha1, 
        old_db.FILE.md5, 
        old_db.FILE.crc32, 
        old_db.FILE.file_name, 
        old_db.FILE.file_size, 
        old_db.FILE.package_id 
    FROM old_db.FILE
    WHERE old_db.FILE.package_id IN (

        SELECT package_id FROM main.PKG
    );
