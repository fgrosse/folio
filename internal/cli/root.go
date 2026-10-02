// Package cli implements the folio command line interface: the root command, its subcommands
// and the database they share.
package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
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

	// BuildVersion is the version this binary was released as, such as "1.0.0". The build of a
	// release sets it, and "folio version" prints it. It is not cobra's Version, which would add
	// a --version flag: folio says its version through a verb, as go does.
	BuildVersion string

	store  *portfolio.SQLiteStore
	quoter portfolio.Quoter // where the prices come from
	now    func() time.Time

	// buildInfo is what the Go toolchain recorded about this binary when it built it, which is
	// where the version comes from if no release build has set one.
	buildInfo func() (*debug.BuildInfo, bool)

	// openTUI runs the interactive views over a store until the user quits. It is the real TUI
	// unless a test puts something in its place, since the real one needs a terminal.
	openTUI func(ctx context.Context, store tui.Store) error
}

// New builds the folio root command with all its subcommands attached.
func New() *Folio {
	cmd := &Folio{
		Command: &cobra.Command{
			Use:   "folio",
			Short: "Track what your stock is worth, held and still to vest",
		},
		quoter:    yahoo.New(),
		now:       time.Now,
		buildInfo: debug.ReadBuildInfo,
	}
	cmd.SilenceErrors = true
	cmd.openTUI = cmd.runProgram

	flags := cmd.PersistentFlags()
	flags.String("db", defaultDBPath(), "path to the folio SQLite database")
	_ = viper.BindPFlags(flags)

	cmd.PersistentPreRunE = cmd.openStore
	cmd.PersistentPostRunE = cmd.closeStore
	cmd.RunE = func(c *cobra.Command, _ []string) error {
		return cmd.openTUI(c.Context(), cmd.store)
	}

	cmd.AddCommand(cmd.LotCmd())
	cmd.AddCommand(cmd.GrantCmd())
	cmd.AddCommand(cmd.ReleaseCmd())
	cmd.AddCommand(cmd.StatusCmd())
	cmd.AddCommand(cmd.DemoCmd())
	cmd.AddCommand(cmd.VersionCmd())

	return cmd
}

// runProgram launches the interactive views over store and returns once the user quits. It hangs
// off the root command rather than a "folio tui" subcommand so that a bare "folio" opens the TUI,
// while status and friends stay plain scriptable verbs - a status bar widget shells out to those
// and must never be handed a full-screen program. Run as the root command's own action, it gets the
// store that PersistentPreRunE opened, and PersistentPostRunE closes it once the program exits. The
// demo runs it over a store of its own.
func (cmd *Folio) runProgram(ctx context.Context, store tui.Store) error {
	model := tui.New(store, cmd.quoter, tui.DefaultStyle())
	program := tea.NewProgram(model, tea.WithContext(ctx))
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
