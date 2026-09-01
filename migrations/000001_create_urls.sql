-- +goose Up
CREATE TABLE urls (
    short_url TEXT PRIMARY KEY,
    full_url TEXT NOT NULL
);

-- +goose Down
DROP TABLE urls;
