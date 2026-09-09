-- +goose Up
ALTER TABLE urls ADD deleted_flag BOOLEAN;

-- +goose Down
ALTER TABLE urls DROP deleted_flag;
