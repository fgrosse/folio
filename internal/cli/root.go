// Package cli implements the folio command line interface: the root command, its subcommands
// and the database they share.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/fgrosse/folio/internal/portfolio"
	"github.com/fgrosse/folio/internal/tui"
	"github.com/fgrosse/folio/internal/yahoo"
)

// Folio is the root command. It owns the store shared by every subcommand: PersistentPreRunE opens
// and migrates it before any subcommand runs, PersistentPostRunE closes it afterwards.
type Folio struct {
	*cobra.Command
	store  *portfolio.SQLiteStore
	quoter portfolio.Quoter // where the prices come from
	now    func() time.Time
}

// New builds the folio root command with all its subcommands attached.
func New() *Folio {
	cmd := &Folio{
		Command: &cobra.Command{
			Use:   "folio",
			Short: "Track what your stock is worth, held and still to vest",
		},
		quoter: yahoo.New(),
		now:    time.Now,
	}
	cmd.SilenceErrors = true

	flags := cmd.PersistentFlags()
	flags.String("db", defaultDBPath(), "path to the folio SQLite database")
	_ = viper.BindPFlags(flags)

	cmd.PersistentPreRunE = cmd.openStore
	cmd.PersistentPostRunE = cmd.closeStore
	cmd.RunE = cmd.runTUI

	cmd.AddCommand(cmd.LotCmd())
	cmd.AddCommand(cmd.GrantCmd())
	cmd.AddCommand(cmd.ReleaseCmd())
	cmd.AddCommand(cmd.StatusCmd())

	return cmd
}

// runTUI launches the interactive views. It hangs off the root command rather than a "folio tui"
// subcommand so that a bare "folio" opens the TUI, while status and friends stay plain scriptable
// verbs - a status bar widget shells out to those and must never be handed a full-screen program.
// Because it runs as the root command's own RunE, it still gets the store that PersistentPreRunE
// opened, and PersistentPostRunE closes it once the program exits.
func (cmd *Folio) runTUI(c *cobra.Command, _ []string) error {
	model := tui.New(cmd.store, cmd.quoter, tui.DefaultStyle())
	program := tea.NewProgram(model, tea.WithContext(c.Context()))
	if _, err := program.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}

	return nil
}

// Println writes to the command's stdout. It exists because cobra.Command's own Print/Println/
// Printf methods write to OutOrStderr, not OutOrStdout - using those directly for regular command
// output would silently send it to stderr and break piping, such as "folio status --json | jq" or
// a status bar widget, which needs the output on stdout.
func (cmd *Folio) Println(a ...any) {
	fmt.Fprintln(cmd.OutOrStdout(), a...)
}

// Printf writes to the command's stdout. See Println for why this isn't cobra.Command.Printf.
func (cmd *Folio) Printf(format string, a ...any) {
	fmt.Fprintf(cmd.OutOrStdout(), format, a...)
}

// defaultDBPath places the database under $XDG_DATA_HOME (or ~/.local/share if unset), following
// the XDG base directory convention for a tool's own persistent data.
func defaultDBPath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".local", "share")
	}

	return filepath.Join(dir, "folio", "folio.db")
}

//goland:noinspection GoResourceLeak
func (cmd *Folio) openStore(*cobra.Command, []string) error {
	viper.SetEnvPrefix("FOLIO")
	viper.AutomaticEnv()

	path := viper.GetString("db")
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("create database directory: %w", err)
		}
	}

	store, err := portfolio.NewStore(path)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	if err := store.Migrate(); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}

	cmd.store = store
	cmd.SilenceUsage = true // past this point, errors are runtime problems, not misuse

	return nil
}

func (cmd *Folio) closeStore(*cobra.Command, []string) error {
	return cmd.store.Close()
}
