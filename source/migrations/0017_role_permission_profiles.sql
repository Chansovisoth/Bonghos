CREATE TABLE role_permission_profiles (
    role        TEXT PRIMARY KEY CHECK (role IN ('admin','member','viewer')),
    customized  INTEGER NOT NULL DEFAULT 0 CHECK (customized IN (0,1)),
    revision    INTEGER NOT NULL DEFAULT 0 CHECK (revision >= 0),
    updated_at  TEXT NOT NULL DEFAULT '',
    updated_by  INTEGER REFERENCES users(id) ON DELETE SET NULL
);

CREATE TABLE role_permissions (
    role        TEXT NOT NULL REFERENCES role_permission_profiles(role) ON DELETE CASCADE,
    permission  TEXT NOT NULL,
    allowed     INTEGER NOT NULL CHECK (allowed IN (0,1)),
    PRIMARY KEY (role, permission)
);
