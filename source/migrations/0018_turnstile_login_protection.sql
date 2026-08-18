CREATE TABLE turnstile_settings (
    id              INTEGER PRIMARY KEY CHECK (id = 1),
    enabled         INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
    site_key        TEXT NOT NULL DEFAULT '',
    secret_key_enc  BLOB,
    updated_at      TEXT NOT NULL DEFAULT '',
    updated_by      INTEGER REFERENCES users(id) ON DELETE SET NULL
);
