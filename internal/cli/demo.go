package cli

import (
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
		Short: "Make up an account to try folio with",
		Long: `
Write a made-up account to a database of its own, to try folio with before entering a real
one: two grants of a company's stock, the shares released from them, one vest that is
waiting to be released, a sale and some stock that was bought.

The account is different every time, unless --seed names one. The stock in it is real, so
folio shows it at today's prices once it has fetched them, and at made-up ones until then.

The database goes to the given path, or next to the one of the real account as demo.db.
Neither is the database of the real account, which the demo leaves alone. A file that is
already there is kept, and the demo refuses to run.
`,
		Example: `
  # Make up an account, and open it
  folio demo
  folio --db ~/.local/share/folio/demo.db

  # The same account every time
  folio demo --seed 7 /tmp/demo.db`,
		Args: cobra.MaximumNArgs(1),
		// The verbs share the database of the real account, which the root command opens and
		// creates before any of them runs. The demo has a database of its own, and must not
		// leave so much as an empty file where the real one goes.
		PersistentPreRunE:  func(*cobra.Command, []string) error { return nil },
		PersistentPostRunE: func(*cobra.Command, []string) error { return nil },
		RunE: func(c *cobra.Command, args []string) error {
			path := defaultDemoPath()
			if len(args) == 1 {
				path = args[0]
			}

			seed, err := c.Flags().GetUint64("seed")
			if err != nil {
				return err
			}
			if !c.Flags().Changed("seed") {
				seed = rand.Uint64()
			}

			c.SilenceUsage = true // past this point, errors are runtime problems, not misuse

			if err := cmd.writeDemo(path, seed); err != nil {
				return err
			}

			cmd.Printf("Wrote a demo account to %s\n\nOpen it with:\n  folio --db %s\n", path, path)
			return nil
		},
	}

	c.Flags().Uint64("seed", 0, "make up the account this number names, rather than a new one every time")

	return c
}

// defaultDemoPath places the demo database next to the one of the real account, under a name that
// says what it is.
func defaultDemoPath() string {
	return filepath.Join(filepath.Dir(defaultDBPath()), "demo.db")
}

// writeDemo makes a new database at path and fills it with the demo account that seed names. A file
// that is at path already is left as it is.
func (cmd *Folio) writeDemo(path string, seed uint64) (err error) {
	if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s exists already: remove it, or give the demo another path", path)
	}

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

	rng := rand.New(rand.NewPCG(seed, seed))
	if err := demo.Fill(store, rng, portfolio.DayOf(cmd.now())); err != nil {
		return fmt.Errorf("make up account: %w", err)
	}

	return nil
}
