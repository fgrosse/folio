-- +migrate Up
CREATE TABLE settings (
	name  TEXT PRIMARY KEY,
	value TEXT NOT NULL
);

-- +migrate Down
DROP TABLE settings;
