-- Lower the monitor scheduling floor from 10s to 5s so ping checks can
-- run as fast as every 5 seconds.
ALTER TABLE monitors DROP CONSTRAINT IF EXISTS monitors_interval_seconds_check;
ALTER TABLE monitors ADD CONSTRAINT monitors_interval_seconds_check CHECK (interval_seconds >= 5);
