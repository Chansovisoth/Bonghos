-- Bonghos initial schema.
-- SQLite stores Bonghos metadata, users, schedules, audit records and
-- operational state. It is never a duplicate source of truth for normal
-- Minecraft files: server directories on disk remain authoritative.

CREATE TABLE users (
    id                INTEGER PRIMARY KEY,
    username          TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash     TEXT NOT NULL,            -- argon2id encoded string
    totp_secret_enc   BLOB,                      -- AES-GCM encrypted TOTP secret
    role              TEXT NOT NULL CHECK (role IN ('owner','admin','member','viewer')),
    disabled          INTEGER NOT NULL DEFAULT 0,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
);

CREATE TABLE recovery_codes (
    id        INTEGER PRIMARY KEY,
    user_id   INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,                     -- sha256 of the one-use code
    used_at   TEXT
);
CREATE INDEX idx_recovery_user ON recovery_codes(user_id);

CREATE TABLE sessions (
    id           TEXT PRIMARY KEY,               -- sha256 of the cookie token
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TEXT NOT NULL,
    expires_at   TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    remote_addr  TEXT NOT NULL DEFAULT '',
    user_agent   TEXT NOT NULL DEFAULT '',
    revoked      INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

CREATE TABLE invitations (
    id          INTEGER PRIMARY KEY,
    token_hash  TEXT NOT NULL UNIQUE,            -- sha256 of the single-use token
    role        TEXT NOT NULL CHECK (role IN ('admin','member','viewer')),
    created_by  INTEGER NOT NULL REFERENCES users(id),
    created_at  TEXT NOT NULL,
    expires_at  TEXT NOT NULL,
    used_by     INTEGER REFERENCES users(id),
    used_at     TEXT,
    revoked     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE login_attempts (
    id          INTEGER PRIMARY KEY,
    identifier  TEXT NOT NULL,                   -- normalized username or ip
    attempted_at TEXT NOT NULL,
    success     INTEGER NOT NULL
);
CREATE INDEX idx_login_attempts ON login_attempts(identifier, attempted_at);

CREATE TABLE instances (
    id                            INTEGER PRIMARY KEY,
    slug                          TEXT NOT NULL,
    display_name                  TEXT NOT NULL,
    server_type                   TEXT NOT NULL DEFAULT 'minecraft-java-modded',
    source_type                   TEXT NOT NULL,
    source_url_host               TEXT NOT NULL DEFAULT '',
    minecraft_version             TEXT NOT NULL DEFAULT '',
    modloader                     TEXT NOT NULL DEFAULT '',
    modloader_version             TEXT NOT NULL DEFAULT '',
    server_directory              TEXT NOT NULL,  -- relative to BONGHOS_HOME unless external
    external_directory            INTEGER NOT NULL DEFAULT 0,
    startup_script                TEXT NOT NULL DEFAULT '',  -- relative to server dir
    java_selection                TEXT NOT NULL DEFAULT 'auto',
    jvm_configuration_source      TEXT NOT NULL DEFAULT '',
    jvm_xms                       TEXT NOT NULL DEFAULT '',
    jvm_xmx                       TEXT NOT NULL DEFAULT '',
    jvm_extra_args                TEXT NOT NULL DEFAULT '',
    icon_revision                 INTEGER NOT NULL DEFAULT 0,
    autostart_enabled             INTEGER NOT NULL DEFAULT 0,
    boot_delay_seconds            INTEGER NOT NULL DEFAULT 30,
    recover_after_unclean_shutdown INTEGER NOT NULL DEFAULT 1,
    restart_policy                TEXT NOT NULL DEFAULT 'on-failure'
                                   CHECK (restart_policy IN ('never','on-failure','always')),
    restart_delay_seconds         INTEGER NOT NULL DEFAULT 10,
    created_at                    TEXT NOT NULL,
    updated_at                    TEXT NOT NULL,
    last_started_at               TEXT,
    last_stopped_at               TEXT,
    UNIQUE (server_type, slug)
);

CREATE TABLE active_instance (
    id          INTEGER PRIMARY KEY CHECK (id = 1),
    instance_id INTEGER REFERENCES instances(id) ON DELETE SET NULL
);
INSERT INTO active_instance (id, instance_id) VALUES (1, NULL);

CREATE TABLE operations (
    id            TEXT PRIMARY KEY,              -- generated operation id
    kind          TEXT NOT NULL,                 -- import_upload / import_url / backup / restore / ...
    instance_id   INTEGER REFERENCES instances(id) ON DELETE SET NULL,
    stage         TEXT NOT NULL,
    status_message TEXT NOT NULL DEFAULT '',
    bytes_processed INTEGER NOT NULL DEFAULT 0,
    total_bytes   INTEGER NOT NULL DEFAULT 0,
    created_by    INTEGER REFERENCES users(id),
    created_at    TEXT NOT NULL,
    started_at    TEXT,
    last_progress_at TEXT,
    finished_at   TEXT,
    error         TEXT NOT NULL DEFAULT '',
    detail_json   TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX idx_operations_stage ON operations(stage);

CREATE TABLE schedules (
    id                  INTEGER PRIMARY KEY,
    instance_id         INTEGER NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    enabled             INTEGER NOT NULL DEFAULT 1,
    timezone            TEXT NOT NULL DEFAULT 'UTC',
    schedule_type       TEXT NOT NULL CHECK (schedule_type IN
                          ('once','hourly','daily','weekly','monthly','fixed_interval','advanced_cron')),
    schedule_expression TEXT NOT NULL,
    action              TEXT NOT NULL CHECK (action IN
                          ('start_server','stop_server','restart_server','send_console_command',
                           'broadcast_message','save_all','create_backup','sequence')),
    action_payload_json TEXT NOT NULL DEFAULT '{}',
    offline_policy      TEXT NOT NULL DEFAULT 'skip_when_offline'
                          CHECK (offline_policy IN ('skip_when_offline','wait_until_online','start_then_execute')),
    missed_run_policy   TEXT NOT NULL DEFAULT 'skip_missed_run'
                          CHECK (missed_run_policy IN ('skip_missed_run','run_once_after_startup')),
    conflict_policy     TEXT NOT NULL DEFAULT 'skip' CHECK (conflict_policy IN ('skip','retry_later')),
    created_by          INTEGER REFERENCES users(id),
    created_at          TEXT NOT NULL,
    updated_at          TEXT NOT NULL,
    next_run_at         TEXT,
    last_run_at         TEXT,
    last_result         TEXT NOT NULL DEFAULT ''
);

CREATE TABLE schedule_runs (
    id          INTEGER PRIMARY KEY,
    schedule_id INTEGER NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    lease_key   TEXT NOT NULL UNIQUE,            -- schedule_id + planned run time: duplicate-run protection
    planned_at  TEXT NOT NULL,
    started_at  TEXT,
    finished_at TEXT,
    status      TEXT NOT NULL DEFAULT 'queued' CHECK (status IN
                  ('queued','running','succeeded','partially_succeeded','failed','skipped','cancelled')),
    detail      TEXT NOT NULL DEFAULT ''
);

CREATE TABLE backups (
    id                 INTEGER PRIMARY KEY,
    backup_id          TEXT NOT NULL UNIQUE,
    instance_id        INTEGER REFERENCES instances(id) ON DELETE SET NULL,
    instance_slug      TEXT NOT NULL,
    display_name       TEXT NOT NULL,
    backup_type        TEXT NOT NULL CHECK (backup_type IN ('full_server','world_and_player_data','configuration_only')),
    consistency_mode   TEXT NOT NULL CHECK (consistency_mode IN ('online','offline')),
    trigger_type       TEXT NOT NULL DEFAULT 'manual',  -- manual/scheduled/safety/emergency_pre_restore
    triggered_by       INTEGER REFERENCES users(id),
    archive_path       TEXT NOT NULL,             -- relative to BONGHOS_HOME
    archive_format     TEXT NOT NULL,
    compressed_size    INTEGER NOT NULL DEFAULT 0,
    uncompressed_size  INTEGER NOT NULL DEFAULT 0,
    file_count         INTEGER NOT NULL DEFAULT 0,
    checksum_algorithm TEXT NOT NULL DEFAULT 'sha256',
    checksum           TEXT NOT NULL DEFAULT '',
    verification_status TEXT NOT NULL DEFAULT 'unverified'
                         CHECK (verification_status IN ('unverified','verified','failed')),
    protected          INTEGER NOT NULL DEFAULT 0,
    minecraft_version  TEXT NOT NULL DEFAULT '',
    modloader          TEXT NOT NULL DEFAULT '',
    modloader_version  TEXT NOT NULL DEFAULT '',
    included_paths     TEXT NOT NULL DEFAULT '[]',
    excluded_paths     TEXT NOT NULL DEFAULT '[]',
    created_at         TEXT NOT NULL,
    completed_at       TEXT
);

CREATE TABLE players (
    id                         INTEGER PRIMARY KEY,
    instance_id                INTEGER NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    username                   TEXT NOT NULL,
    uuid                       TEXT NOT NULL DEFAULT '',
    first_seen_at              TEXT NOT NULL,
    last_seen_at               TEXT NOT NULL,
    last_joined_at             TEXT,
    last_left_at               TEXT,
    current_session_started_at TEXT,
    observed_playtime_seconds  INTEGER NOT NULL DEFAULT 0,
    is_online                  INTEGER NOT NULL DEFAULT 0,
    UNIQUE (instance_id, username)
);

CREATE TABLE metrics (
    id            INTEGER PRIMARY KEY,
    collected_at  TEXT NOT NULL,
    instance_id   INTEGER,
    cpu_percent   REAL NOT NULL DEFAULT 0,
    rss_bytes     INTEGER NOT NULL DEFAULT 0,
    host_mem_total INTEGER NOT NULL DEFAULT 0,
    host_mem_avail INTEGER NOT NULL DEFAULT 0,
    load1         REAL NOT NULL DEFAULT 0,
    disk_total    INTEGER NOT NULL DEFAULT 0,
    disk_free     INTEGER NOT NULL DEFAULT 0,
    server_dir_bytes INTEGER NOT NULL DEFAULT 0,
    backup_dir_bytes INTEGER NOT NULL DEFAULT 0,
    online_players INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX idx_metrics_time ON metrics(collected_at);

CREATE TABLE audit_log (
    id         INTEGER PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    user_id    INTEGER,
    username   TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL,
    target     TEXT NOT NULL DEFAULT '',
    detail     TEXT NOT NULL DEFAULT '',
    remote_addr TEXT NOT NULL DEFAULT ''
);
CREATE INDEX idx_audit_time ON audit_log(occurred_at);

CREATE TABLE server_sessions (
    id           INTEGER PRIMARY KEY,
    instance_id  INTEGER NOT NULL REFERENCES instances(id) ON DELETE CASCADE,
    started_at   TEXT NOT NULL,
    ended_at     TEXT,
    exit_code    INTEGER,
    term_signal  TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL DEFAULT 'unknown'
                   CHECK (classification IN ('unknown','clean_stop','requested_restart','crash','supervisor_failure','unclean'))
);

CREATE TABLE retention_policies (
    id             INTEGER PRIMARY KEY,
    instance_id    INTEGER REFERENCES instances(id) ON DELETE CASCADE,
    max_count      INTEGER NOT NULL DEFAULT 0,   -- 0 = unlimited
    max_age_days   INTEGER NOT NULL DEFAULT 0,
    max_storage_mb INTEGER NOT NULL DEFAULT 0,
    keep_daily     INTEGER NOT NULL DEFAULT 0,
    keep_weekly    INTEGER NOT NULL DEFAULT 0,
    keep_monthly   INTEGER NOT NULL DEFAULT 0
);
