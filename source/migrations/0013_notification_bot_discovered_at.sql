ALTER TABLE notification_bot_discoveries ADD COLUMN discovered_at TEXT NOT NULL DEFAULT '';

-- Older installations only recorded when the destination was last observed.
-- Preserve that timestamp as the best available first-discovery estimate.
UPDATE notification_bot_discoveries
SET discovered_at = last_seen_at
WHERE discovered_at = '';
