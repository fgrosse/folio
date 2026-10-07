-- +migrate Up
ALTER TABLE sales ADD COLUMN cleared BOOLEAN NOT NULL DEFAULT FALSE;

-- +migrate Down
ALTER TABLE sales DROP COLUMN cleared;
