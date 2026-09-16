-- +goose Up
ALTER TABLE urls ADD user_id TEXT;
CREATE INDEX user_id_idx ON urls (user_id);

-- +goose Down
ALTER TABLE urls DROP user_id;
DROP INDEX IF EXISTS user_id_idx;
