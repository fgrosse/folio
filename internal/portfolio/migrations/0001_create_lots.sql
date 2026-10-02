-- +migrate Up
CREATE TABLE lots (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	symbol      TEXT NOT NULL,
	shares      TEXT NOT NULL,
	acquired_on DATE NOT NULL
);

-- +migrate Down
DROP TABLE lots;
