-- +goose Up
CREATE TABLE tenants (
    id         TEXT PRIMARY KEY,
    name       TEXT    NOT NULL,
    slug       TEXT    NOT NULL UNIQUE,
    status     TEXT    NOT NULL DEFAULT 'creating'
        CHECK (status IN ('creating', 'active', 'suspended', 'deleting', 'deleted')),
    plan       TEXT    NOT NULL DEFAULT 'free',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_tenants_status ON tenants (status);
CREATE INDEX idx_tenants_slug   ON tenants (slug);

-- +goose Down
DROP TABLE IF EXISTS tenants;
