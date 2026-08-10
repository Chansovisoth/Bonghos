-- Encrypted Telegram and Discord notification bot configuration.
CREATE TABLE notification_bots (
    id                    INTEGER PRIMARY KEY,
    name                  TEXT NOT NULL,
    provider              TEXT NOT NULL CHECK (provider IN ('telegram','discord')),
    token_enc             BLOB NOT NULL,
    destination_id        TEXT NOT NULL,
    enabled               INTEGER NOT NULL DEFAULT 1,
    notify_server_started INTEGER NOT NULL DEFAULT 1,
    notify_server_stopped INTEGER NOT NULL DEFAULT 1,
    notify_player_joined  INTEGER NOT NULL DEFAULT 1,
    notify_player_left    INTEGER NOT NULL DEFAULT 1,
    created_at            TEXT NOT NULL,
    updated_at            TEXT NOT NULL
);

CREATE INDEX idx_notification_bots_enabled ON notification_bots(enabled, provider);
