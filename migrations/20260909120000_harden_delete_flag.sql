-- +goose Up
UPDATE urls SET deleted_flag = false WHERE deleted_flag IS NULL;
ALTER TABLE urls ALTER COLUMN deleted_flag SET DEFAULT false;
ALTER TABLE urls ALTER COLUMN deleted_flag SET NOT NULL;

-- +goose Down
ALTER TABLE urls ALTER COLUMN deleted_flag DROP NOT NULL;
ALTER TABLE urls ALTER COLUMN deleted_flag DROP DEFAULT;
