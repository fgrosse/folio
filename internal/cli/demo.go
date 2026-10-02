package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/demo"
	"github.com/fgrosse/folio/internal/portfolio"
)

// DemoCmd returns the "folio demo" command.
func (cmd *Folio) DemoCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "demo [path]",
		Short: "Try folio with a made-up account",
		Long: `
Open folio on a made-up account, to try it with before entering a real one: two grants of a
company's stock, the shares released from them, one vest that is waiting to be released, a
sale and some stock that was bought.

The account is different every time, unless --seed names one. The stock in it is real, so
folio shows it at today's prices once it has fetched them, and at made-up ones until then.

The account is gone again when folio is closed. To keep it, give a path: the demo writes its
database there and leaves it, and opens what is there the next time it is given that path,
with whatever was changed since.

The real account is never part of this. The demo has a database of its own, and does not
open the one that folio otherwise uses.
`,
		Example: `
  # Look around
  folio demo

  # The same account every time
  folio demo --seed 7

  # A demo account to come back to
  folio demo ~/folio-demo.db`,
		Args: cobra.MaximumNArgs(1),
		// The verbs share the database of the real account, which the root command opens and
		// creates before any of them runs. The demo has a database of its own, and must not
		// leave so much as an empty file where the real one goes.
		PersistentPreRunE:  func(*cobra.Command, []string) error { return nil },
		PersistentPostRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(c *cobra.Command, args []string) error {
			seed, err := c.Flags().GetUint64("seed")
			if err != nil {
				return err
			}
			if !c.Flags().Changed("seed") {
				seed = rand.Uint64()
			}

			c.SilenceUsage = true // past this point, errors are runtime problems, not misuse

			if len(args) == 1 {
				return cmd.runDemo(c.Context(), args[0], seed)
			}

			// Without a path to keep it at, the account lives in a directory that goes when the
			// TUI does. A directory rather than a file, since SQLite keeps files of its own next
			// to the database.
			dir, err := os.MkdirTemp("", "folio-demo-")
			if err != nil {
				return fmt.Errorf("create database directory: %w", err)
			}
			defer func() { _ = os.RemoveAll(dir) }()

			return cmd.runDemo(c.Context(), filepath.Join(dir, "demo.db"), seed)
		},
	}

	c.Flags().Uint64("seed", 0, "make up the account this number names, rather than a new one every time")

	return c
}

// runDemo opens the TUI on the demo database at path and returns once the user quits. If there is
// no database at path yet it makes one first, with the demo account that seed names. One that is
// there is opened as it is.
func (cmd *Folio) runDemo(ctx context.Context, path string, seed uint64) (err error) {
	_, statErr := os.Stat(path)
	isNew := errors.Is(statErr, fs.ErrNotExist)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create database directory: %w", err)
	}

	store, err := portfolio.NewStore(path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer func() { err = errors.Join(err, store.Close()) }()

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	if isNew {
		rng := rand.New(rand.NewPCG(seed, seed))
		if err := demo.Fill(store, rng, portfolio.DayOf(cmd.now())); err != nil {
			return fmt.Errorf("make up account: %w", err)
		}
	}

	return cmd.openTUI(ctx, store)
}
