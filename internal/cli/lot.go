package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// LotCmd returns the "folio lot" command.
func (cmd *Folio) LotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lot <shares> <symbol> [YYYY-MM-DD]",
		Short: "Record shares that you hold",
		Long: `
Record a lot: a number of shares of one stock that you hold and that all arrived on the
same day, such as what one order bought. The lots are what the current account value counts.

The day is the one the shares arrived on, and today if it is left out. Quoting is optional:
any positional arguments are joined with spaces.
`,
		Example: `
  # Record 12.5 shares that arrived today
  folio lot 12.5 PANW

  # Record shares that arrived on a day in the past
  folio lot 40 PANW 2026-03-15`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			lot, err := portfolio.NewLot(strings.Join(args, " "))
			if err != nil {
				return err
			}

			return cmd.store.SaveLot(lot)
		},
	}
}
