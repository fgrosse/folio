-- +migrate Up
ALTER TABLE lots ADD COLUMN cost TEXT;

-- +migrate Down
ALTER TABLE lots DROP COLUMN cost;
