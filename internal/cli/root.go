// Package cli implements the folio command line interface: the root command, its subcommands
// and the database they share.
package cli

import (
	"github.com/spf13/cobra"
)

// Folio is the root command.
type Folio struct {
	*cobra.Command
}

// New builds the folio root command with all its subcommands attached.
func New() *Folio {
	cmd := &Folio{
		Command: &cobra.Command{
			Use:   "folio",
			Short: "Track what your stock is worth, held and still to vest",
		},
	}
	cmd.SilenceErrors = true

	return cmd
}
