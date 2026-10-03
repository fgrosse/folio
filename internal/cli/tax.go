package cli

import (
	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TaxRateCmd returns the "folio tax-rate" command.
func (cmd *Folio) TaxRateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tax-rate <percent>",
		Short: "Set the rate that your vests are taxed at",
		Long: `
Set the rate that the shares still to vest are taxed at, as a percentage. A vest is
taxed as income when it vests, at a rate that depends on the rest of the year's
income and on where you live, so folio does not work it out. It takes one rate
for every vest instead: your estimate of the rate at the top of your income.

The Vesting view shows what each vest is worth after tax at this rate.
`,
		Example: `
  # Have vests taxed at 44.3%
  folio tax-rate 44.3%`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			rate, err := portfolio.ParseTaxRate(args[0])
			if err != nil {
				return err
			}

			return cmd.store.SetTaxRate(rate)
		},
	}
}
