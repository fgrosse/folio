-- +migrate Up
ALTER TABLE lots ADD COLUMN vest_id INTEGER REFERENCES vests (id);

-- +migrate Down
ALTER TABLE lots DROP COLUMN vest_id;
