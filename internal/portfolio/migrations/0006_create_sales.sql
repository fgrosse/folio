-- +migrate Up
CREATE TABLE sales (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	lot_id  INTEGER NOT NULL REFERENCES lots (id),
	sold_on DATE    NOT NULL,
	shares  TEXT    NOT NULL,
	price   TEXT    NOT NULL,
	note    TEXT    NOT NULL DEFAULT ''
);

-- +migrate Down
DROP TABLE sales;
