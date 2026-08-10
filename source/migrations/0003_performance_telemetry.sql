-- Rich performance history for the operator Performance page.
ALTER TABLE metrics ADD COLUMN host_cpu_percent REAL NOT NULL DEFAULT 0;
ALTER TABLE metrics ADD COLUMN cpu_temp_celsius REAL;
ALTER TABLE metrics ADD COLUMN jvm_xms_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metrics ADD COLUMN jvm_xmx_bytes INTEGER NOT NULL DEFAULT 0;
