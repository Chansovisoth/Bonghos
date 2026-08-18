CREATE TABLE playit_settings (
    id                  INTEGER PRIMARY KEY CHECK (id = 1),
    enabled             INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0,1)),
    account_mode        TEXT NOT NULL DEFAULT 'account' CHECK (account_mode IN ('account','guest')),
    management_mode     TEXT NOT NULL DEFAULT 'none' CHECK (management_mode IN ('none','external','bonghos')),
    agent_secret_enc    BLOB,
    agent_id            TEXT NOT NULL DEFAULT '',
    tunnel_id           TEXT NOT NULL DEFAULT '',
    public_address      TEXT NOT NULL DEFAULT '',
    local_port          INTEGER NOT NULL DEFAULT 25565 CHECK (local_port BETWEEN 1 AND 65535),
    claim_code          TEXT NOT NULL DEFAULT '',
    claim_started_at    TEXT NOT NULL DEFAULT '',
    updated_at          TEXT NOT NULL DEFAULT '',
    updated_by          INTEGER REFERENCES users(id) ON DELETE SET NULL
);

-- Updating an existing Bonghos installation must preserve its current
-- networking behavior. Playit remains disabled until an Owner or explicitly
-- delegated administrator opts in from Settings.
INSERT INTO playit_settings (id, enabled, account_mode, management_mode, local_port)
VALUES (1, 0, 'account', 'none', 25565);
