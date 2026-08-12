-- Allow one notification bot credential to target multiple approved chats.
CREATE TABLE notification_bot_destinations (
    bot_id         INTEGER NOT NULL REFERENCES notification_bots(id) ON DELETE CASCADE,
    destination_id TEXT NOT NULL,
    display_name   TEXT NOT NULL DEFAULT '',
    destination_type TEXT NOT NULL DEFAULT '',
    photo_file_id  TEXT NOT NULL DEFAULT '',
    thread_id      INTEGER NOT NULL DEFAULT 0,
    thread_name    TEXT NOT NULL DEFAULT '',
    position       INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (bot_id, destination_id)
);

INSERT INTO notification_bot_destinations
    (bot_id, destination_id, display_name, destination_type, position)
SELECT id, destination_id, '', '', 0
FROM notification_bots
WHERE trim(destination_id) <> '';

CREATE INDEX idx_notification_bot_destinations_bot
    ON notification_bot_destinations(bot_id, position);

-- Keep every Telegram group/topic the owner has discovered, independently
-- from the destinations currently enabled for broadcasts.
CREATE TABLE notification_bot_discoveries (
    bot_id           INTEGER NOT NULL REFERENCES notification_bots(id) ON DELETE CASCADE,
    destination_id   TEXT NOT NULL,
    display_name     TEXT NOT NULL DEFAULT '',
    destination_type TEXT NOT NULL DEFAULT '',
    photo_file_id    TEXT NOT NULL DEFAULT '',
    is_forum         INTEGER NOT NULL DEFAULT 0,
    topics_json      TEXT NOT NULL DEFAULT '[]',
    last_seen_at     TEXT NOT NULL,
    PRIMARY KEY (bot_id, destination_id)
);

CREATE INDEX idx_notification_bot_discoveries_bot
    ON notification_bot_discoveries(bot_id, display_name COLLATE NOCASE);

-- Preserve existing installations even if they already contain more bots,
-- while preventing any further additions that exceed the supported layout.
CREATE TRIGGER notification_bots_limit_insert
BEFORE INSERT ON notification_bots
WHEN (SELECT COUNT(*) FROM notification_bots) >= 2
  OR EXISTS (SELECT 1 FROM notification_bots WHERE provider=NEW.provider)
BEGIN
    SELECT RAISE(ABORT, 'only one Telegram bot and one Discord bot are supported');
END;
