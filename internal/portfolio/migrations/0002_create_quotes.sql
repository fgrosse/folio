-- +migrate Up
CREATE TABLE quotes (
	symbol         TEXT     PRIMARY KEY,
	price          TEXT     NOT NULL,
	previous_close TEXT     NOT NULL,
	currency       TEXT     NOT NULL,
	quoted_at      DATETIME NOT NULL
);

-- +migrate Down
DROP TABLE quotes;
