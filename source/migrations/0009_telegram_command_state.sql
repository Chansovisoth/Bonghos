-- Track the last Telegram update consumed by Bonghos. This prevents old
-- /bonghos here commands from being replayed after a restart or destination change.
CREATE TABLE notification_bot_telegram_state (
    bot_id         INTEGER PRIMARY KEY REFERENCES notification_bots(id) ON DELETE CASCADE,
    last_update_id INTEGER NOT NULL DEFAULT 0
);
