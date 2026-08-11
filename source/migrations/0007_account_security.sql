ALTER TABLE recovery_codes ADD COLUMN created_at TEXT NOT NULL DEFAULT '';

UPDATE recovery_codes
SET created_at = COALESCE(
    (SELECT created_at FROM users WHERE users.id = recovery_codes.user_id),
    strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
)
WHERE created_at = '';
