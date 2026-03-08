-- +goose Up
ALTER TABLE users ADD COLUMN theme TEXT NOT NULL DEFAULT 'system'
    CHECK (theme IN ('light', 'dark', 'system'));
ALTER TABLE users ADD COLUMN language TEXT NOT NULL DEFAULT 'en'
    CHECK (language IN ('en', 'es'));

-- +goose Down
ALTER TABLE users DROP COLUMN theme;
ALTER TABLE users DROP COLUMN language;
