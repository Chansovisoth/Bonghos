-- Bonghos home size and its system-files portion for storage breakdown charts.
ALTER TABLE metrics ADD COLUMN bonghos_dir_bytes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE metrics ADD COLUMN system_dir_bytes INTEGER NOT NULL DEFAULT 0;
