-- Set the output mode to CSV (comma-separated)
.mode csv

-- Tell SQLite to send all future output to this file
.output package_counts.csv

SELECT package_id, COUNT(*) AS file_count 
FROM old_db.FILE 
GROUP BY package_id 
ORDER BY file_count DESC;