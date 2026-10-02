// Package portfolio holds folio's domain logic: the lots, grants and vests an account is made of,
// what they are worth, and the SQLite store that persists them.
package portfolio

import (
	"embed"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // the database driver
	migrate "github.com/rubenv/sql-migrate"
	"github.com/shopspring/decimal"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// migrations are the schema migrations embedded in the binary, in the order they apply.
var migrations = migrate.EmbedFileSystemMigrationSource{
	FileSystem: migrationsFS,
	Root:       "migrations",
}

// SQLiteStore is a Store backed by a SQLite database.
type SQLiteStore struct {
	db *sqlx.DB
}

// NewStore opens a SQLite database at dsn - a file path, or ":memory:" for a database that only
// ever lives in memory. It does not touch the schema; call Migrate explicitly once the store is
// ready to be used against a fresh or outdated database. The returned store keeps a single
// connection open: SQLite only supports one writer at a time, so pooling multiple connections
// buys nothing and would, for an in-memory dsn, silently scatter data across unrelated empty
// databases instead of one shared one.
func NewStore(dsn string) (*SQLiteStore, error) {
	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// Migrate applies every pending schema migration embedded in the binary; it is a no-op if the
// database is already current. NewStore never calls this on its own - the portfolio package leaves
// it up to the caller, and the folio CLI calls it on every startup, which is enough to create a
// fresh database and keep an existing one current without a separate step.
func (s *SQLiteStore) Migrate() error {
	_, err := migrate.Exec(s.db.DB, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// Lots lists every lot.
func (s *SQLiteStore) Lots() ([]Lot, error) {
	var rows []struct {
		ID       int             `db:"id"`
		Symbol   string          `db:"symbol"`
		Shares   decimal.Decimal `db:"shares"`
		Acquired time.Time       `db:"acquired_on"`
	}

	if err := s.db.Select(&rows, `SELECT id, symbol, shares, acquired_on FROM lots`); err != nil {
		return nil, err
	}

	lots := make([]Lot, len(rows))
	for i, r := range rows {
		lots[i] = Lot{ID: r.ID, Symbol: r.Symbol, Shares: r.Shares, Acquired: r.Acquired}
	}

	return lots, nil
}

// SaveLot records a new lot.
func (s *SQLiteStore) SaveLot(lot Lot) error {
	_, err := s.db.Exec(
		`INSERT INTO lots (symbol, shares, acquired_on) VALUES (?, ?, ?)`,
		lot.Symbol, lot.Shares, lot.Acquired.Format(time.DateOnly),
	)

	return err
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
