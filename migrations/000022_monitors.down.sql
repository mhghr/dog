ALTER TABLE monitors DROP COLUMN severity;
ALTER TABLE monitors DROP COLUMN monitor_type_id;
ALTER TABLE monitors DROP COLUMN resource_id;

DROP TABLE IF EXISTS monitor_jobs;
DROP TYPE IF EXISTS job_status;

DROP TABLE IF EXISTS probe_assignments;
DROP TABLE IF EXISTS monitor_types;
