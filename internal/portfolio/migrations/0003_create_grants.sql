-- +migrate Up
CREATE TABLE grants (
	id     INTEGER PRIMARY KEY AUTOINCREMENT,
	name   TEXT NOT NULL,
	symbol TEXT NOT NULL
);

CREATE TABLE vests (
	id       INTEGER PRIMARY KEY AUTOINCREMENT,
	grant_id INTEGER NOT NULL REFERENCES grants (id),
	vests_on DATE    NOT NULL,
	shares   TEXT    NOT NULL
);

-- +migrate Down
DROP TABLE vests;
DROP TABLE grants;
