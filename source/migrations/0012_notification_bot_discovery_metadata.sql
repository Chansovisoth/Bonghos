-- Track provider containers independently from configured broadcast targets.
-- Telegram discoveries represent groups; Discord discoveries represent servers.
ALTER TABLE notification_bot_discoveries ADD COLUMN guild_id TEXT NOT NULL DEFAULT '';
ALTER TABLE notification_bot_discoveries ADD COLUMN guild_name TEXT NOT NULL DEFAULT '';
ALTER TABLE notification_bot_discoveries ADD COLUMN guild_icon TEXT NOT NULL DEFAULT '';
