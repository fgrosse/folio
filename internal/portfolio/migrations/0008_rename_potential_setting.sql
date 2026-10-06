-- +migrate Up
-- The settings "potential" and "current", each gross or net, became the one setting
-- "show-net-summary", true or false. An account that showed its potential value net goes on
-- showing its values net. "current" was never in a release and is dropped with it.
INSERT OR IGNORE INTO settings (name, value)
SELECT 'show-net-summary', 'true' FROM settings WHERE name = 'potential' AND value = 'net';

DELETE FROM settings WHERE name IN ('potential', 'current');

-- +migrate Down
INSERT OR IGNORE INTO settings (name, value)
SELECT 'potential', 'net' FROM settings WHERE name = 'show-net-summary' AND value = 'true';

DELETE FROM settings WHERE name = 'show-net-summary';
