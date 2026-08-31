-- +goose Up
CREATE UNIQUE INDEX full_url ON urls (full_url);

-- +goose Down
DROP INDEX IF EXISTS full_url;
