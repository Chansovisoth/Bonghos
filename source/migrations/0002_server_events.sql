-- Server lifecycle events.
--
-- Until now "what is happening" was scattered across supervisor-state.json
-- (current state only), transient WebSocket broadcasts (lost if no browser was
-- open), the metrics table (numbers, no narrative) and audit_log (user actions,
-- not what the server did on its own). Answering "why did it restart at 3am"
-- meant reading journalctl and latest.log by hand.
--
-- This table is the durable timeline: a small, bounded, append-only record of
-- what the server did, suitable for showing directly in the interface.

CREATE TABLE server_events (
    id           INTEGER PRIMARY KEY,
    occurred_at  TEXT    NOT NULL,
    instance_id  INTEGER REFERENCES instances(id) ON DELETE CASCADE,

    -- Coarse category, used for filtering and colouring:
    -- lifecycle | progress | backup | schedule | recovery | error
    category     TEXT    NOT NULL,

    -- Specific event, e.g. starting, java_started, loading_mods, ready,
    -- stopping, stopped, crashed, restarting, restart_scheduled,
    -- backup_started, backup_completed, restore_completed, boot_recovery.
    event        TEXT    NOT NULL,

    -- info | warning | error
    severity     TEXT    NOT NULL DEFAULT 'info',

    -- Human-readable line shown in the interface.
    message      TEXT    NOT NULL DEFAULT '',

    -- Optional structured extras as JSON (exit code, pid, duration, backup id).
    detail       TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_server_events_time ON server_events(occurred_at DESC);
CREATE INDEX idx_server_events_instance ON server_events(instance_id, id DESC);
