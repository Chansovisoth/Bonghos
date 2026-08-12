-- Preserve Discord server identity alongside each selected channel.
ALTER TABLE notification_bot_destinations ADD COLUMN guild_id TEXT NOT NULL DEFAULT '';
ALTER TABLE notification_bot_destinations ADD COLUMN guild_name TEXT NOT NULL DEFAULT '';
ALTER TABLE notification_bot_destinations ADD COLUMN guild_icon TEXT NOT NULL DEFAULT '';
