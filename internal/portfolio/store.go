// Package portfolio holds folio's domain logic: the lots, grants and vests an account is made of,
// what they are worth, and the SQLite store that persists them.
package portfolio

import (
	"database/sql"
	"embed"
	"errors"
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
// databases instead of one shared one. That one connection is also why the settings NewStore makes
// on it hold for as long as the store is open.
func NewStore(dsn string) (*SQLiteStore, error) {
	db, err := sqlx.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)

	// The TUI and a status bar widget open the same file at the same time, and both save quotes. In
	// WAL mode a reader does not block a writer, and with a busy timeout a writer waits for the
	// other to finish instead of failing at once with "database is locked".
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA busy_timeout = 5000`); err != nil {
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

// Lots lists every lot in the order they were acquired, and lots of the same day in the order they
// were saved. A lot that a vest was released into comes with the name of the vest's grant.
func (s *SQLiteStore) Lots() ([]Lot, error) {
	var rows []struct {
		ID       int                 `db:"id"`
		Symbol   string              `db:"symbol"`
		Shares   decimal.Decimal     `db:"shares"`
		Acquired time.Time           `db:"acquired_on"`
		Cost     decimal.NullDecimal `db:"cost"`
		Grant    string              `db:"grant_name"`
	}

	err := s.db.Select(&rows, `
		SELECT lots.id, lots.symbol, lots.shares, lots.acquired_on, lots.cost, coalesce(grants.name, '') AS grant_name
		FROM lots
			LEFT JOIN vests ON vests.id = lots.vest_id
			LEFT JOIN grants ON grants.id = vests.grant_id
		ORDER BY lots.acquired_on, lots.id`)
	if err != nil {
		return nil, err
	}

	lots := make([]Lot, len(rows))
	for i, r := range rows {
		lots[i] = Lot{ID: r.ID, Symbol: r.Symbol, Shares: r.Shares, Acquired: r.Acquired, Cost: r.Cost.Decimal, Grant: r.Grant}
	}

	return lots, nil
}

// SaveLot records a new lot, which has to be valid.
func (s *SQLiteStore) SaveLot(lot Lot) error {
	if err := lot.Validate(); err != nil {
		return err
	}

	_, err := s.db.Exec(
		`INSERT INTO lots (symbol, shares, acquired_on, cost) VALUES (?, ?, ?, ?)`,
		lot.Symbol, lot.Shares, lot.Acquired.Format(time.DateOnly), nullable(lot.Cost),
	)

	return err
}

// nullable is cost the way the database holds it: NULL for a cost of zero, which stands for a cost
// that is not known, so that it does not read as shares that were free.
func nullable(cost decimal.Decimal) decimal.NullDecimal {
	return decimal.NullDecimal{Decimal: cost, Valid: !cost.IsZero()}
}

// DeleteLot deletes the lot with the given ID, which has to exist.
func (s *SQLiteStore) DeleteLot(id int) error {
	result, err := s.db.Exec(`DELETE FROM lots WHERE id = ?`, id)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no lot with ID %d", id)
	}

	return nil
}

// Grants lists every grant in the order they were saved, each with its vests in the order of
// their days.
func (s *SQLiteStore) Grants() ([]Grant, error) {
	var grants []Grant
	if err := s.db.Select(&grants, `SELECT id, name, symbol FROM grants ORDER BY id`); err != nil {
		return nil, err
	}

	// A vest is released once a lot points at it.
	var rows []struct {
		ID       int             `db:"id"`
		GrantID  int             `db:"grant_id"`
		Date     time.Time       `db:"vests_on"`
		Shares   decimal.Decimal `db:"shares"`
		Released bool            `db:"released"`
	}
	err := s.db.Select(&rows, `
		SELECT id, grant_id, vests_on, shares, EXISTS (SELECT 1 FROM lots WHERE lots.vest_id = vests.id) AS released
		FROM vests
		ORDER BY vests_on, id`)
	if err != nil {
		return nil, err
	}

	for i := range grants {
		for _, r := range rows {
			if r.GrantID == grants[i].ID {
				vest := Vest{ID: r.ID, Date: r.Date, Shares: r.Shares, Released: r.Released}
				grants[i].Vests = append(grants[i].Vests, vest)
			}
		}
	}

	return grants, nil
}

// SaveGrant records a new grant together with its vests, all of them or none.
func (s *SQLiteStore) SaveGrant(grant Grant) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // does nothing once the transaction is committed

	result, err := tx.Exec(`INSERT INTO grants (name, symbol) VALUES (?, ?)`, grant.Name, grant.Symbol)
	if err != nil {
		return err
	}

	grantID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	for _, vest := range grant.Vests {
		_, err := tx.Exec(
			`INSERT INTO vests (grant_id, vests_on, shares) VALUES (?, ?, ?)`,
			grantID, vest.Date.Format(time.DateOnly), vest.Shares,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// DeleteGrant deletes the grant with the given ID, which has to exist, together with its vests. The
// lots its vests were released into stay, and no longer say which vest they came from.
func (s *SQLiteStore) DeleteGrant(id int) error {
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // does nothing once the transaction is committed

	_, err = tx.Exec(`UPDATE lots SET vest_id = NULL WHERE vest_id IN (SELECT id FROM vests WHERE grant_id = ?)`, id)
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DELETE FROM vests WHERE grant_id = ?`, id); err != nil {
		return err
	}

	result, err := tx.Exec(`DELETE FROM grants WHERE id = ?`, id)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("no grant with ID %d", id)
	}

	return tx.Commit()
}

// ReleaseVest turns the vest with the given ID into a lot of its grant's stock, acquired on the day
// of the vest and holding shares, the number that actually arrived: more than none, and no more than
// vested. cost is what one share was worth that day, which is what the lot cost, or zero if that is
// not known. The vest counts as released from then on, for as long as that lot exists, and cannot be
// released a second time.
func (s *SQLiteStore) ReleaseVest(id int, shares, cost decimal.Decimal) error {
	if !shares.IsPositive() {
		return errors.New("a vest must release more than 0 shares")
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }() // does nothing once the transaction is committed

	var vest struct {
		Date     time.Time       `db:"vests_on"`
		Shares   decimal.Decimal `db:"shares"`
		Released bool            `db:"released"`
	}
	err = tx.Get(&vest, `
		SELECT vests_on, shares, EXISTS (SELECT 1 FROM lots WHERE lots.vest_id = vests.id) AS released
		FROM vests
		WHERE id = ?`, id)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("no vest with ID %d", id)
	case err != nil:
		return err
	case vest.Released:
		return fmt.Errorf("the vest of %s is released already", vest.Date.Format(time.DateOnly))
	case shares.GreaterThan(vest.Shares):
		return fmt.Errorf("the vest of %s has %s shares, not %s", vest.Date.Format(time.DateOnly), vest.Shares, shares)
	}

	_, err = tx.Exec(`
		INSERT INTO lots (symbol, shares, acquired_on, cost, vest_id)
		SELECT grants.symbol, ?, vests.vests_on, ?, vests.id
		FROM vests JOIN grants ON grants.id = vests.grant_id
		WHERE vests.id = ?`,
		shares, nullable(cost), id,
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Quotes returns the latest quote the database has of every symbol it has one of, by symbol. The
// database is the cache of the prices: what comes back is as old as the last SaveQuote.
func (s *SQLiteStore) Quotes() (map[string]Quote, error) {
	var rows []struct {
		Symbol        string          `db:"symbol"`
		Price         decimal.Decimal `db:"price"`
		PreviousClose decimal.Decimal `db:"previous_close"`
		Currency      string          `db:"currency"`
		At            time.Time       `db:"quoted_at"`
	}

	if err := s.db.Select(&rows, `SELECT symbol, price, previous_close, currency, quoted_at FROM quotes`); err != nil {
		return nil, err
	}

	quotes := make(map[string]Quote, len(rows))
	for _, r := range rows {
		quotes[r.Symbol] = Quote{
			Symbol:        r.Symbol,
			Price:         r.Price,
			PreviousClose: r.PreviousClose,
			Currency:      r.Currency,
			At:            r.At,
		}
	}

	return quotes, nil
}

// SaveQuote records quote as the latest of its symbol, in place of the one before. Its time is
// stored to the whole second and in UTC.
func (s *SQLiteStore) SaveQuote(quote Quote) error {
	_, err := s.db.Exec(`
		INSERT INTO quotes (symbol, price, previous_close, currency, quoted_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT (symbol) DO UPDATE SET
			price = excluded.price,
			previous_close = excluded.previous_close,
			currency = excluded.currency,
			quoted_at = excluded.quoted_at`,
		quote.Symbol, quote.Price, quote.PreviousClose, quote.Currency, quote.At.UTC().Truncate(time.Second),
	)

	return err
}

// Close closes the database.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
