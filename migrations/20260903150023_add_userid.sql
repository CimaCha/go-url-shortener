-- +goose Up
ALTER TABLE urls ADD user_id TEXT;

-- +goose Down
ALTER TABLE urls DROP user_id;
