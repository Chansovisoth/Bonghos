-- Telegram destinations created by the former discovery/dropdown flow cannot
-- be distinguished reliably from deleted forum topics. Reset them once so
-- every destination shown after this upgrade was explicitly connected with
-- /bonghos here under the command-based flow.
DELETE FROM notification_bot_destinations
WHERE bot_id IN (SELECT id FROM notification_bots WHERE provider='telegram');

DELETE FROM notification_bot_discoveries
WHERE bot_id IN (SELECT id FROM notification_bots WHERE provider='telegram');

DELETE FROM notification_bot_telegram_state
WHERE bot_id IN (SELECT id FROM notification_bots WHERE provider='telegram');

UPDATE notification_bots
SET destination_id='', updated_at=strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE provider='telegram';
