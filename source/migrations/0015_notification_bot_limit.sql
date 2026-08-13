DROP TRIGGER IF EXISTS notification_bots_limit_insert;

-- Preserve existing installations while preventing additions beyond two bots
-- per provider or four notification bots total.
CREATE TRIGGER notification_bots_limit_insert
BEFORE INSERT ON notification_bots
WHEN (SELECT COUNT(*) FROM notification_bots) >= 4
  OR (SELECT COUNT(*) FROM notification_bots WHERE provider=NEW.provider) >= 2
BEGIN
    SELECT RAISE(ABORT, 'only two Telegram bots and two Discord bots are supported');
END;
