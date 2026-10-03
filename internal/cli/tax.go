package cli

import (
	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TaxRateCmd returns the "folio tax-rate" command.
func (cmd *Folio) TaxRateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tax-rate [percent]",
		Short: "Set the rate that your vests are taxed at",
		Long: `
Set the rate that the shares still to vest are taxed at, as a percentage. A vest is
taxed as income when it vests, at a rate that depends on the rest of the year's
income and on where you live, so folio does not work it out. It takes one rate
for every vest instead: your estimate of the rate at the top of your income.

The Vesting view shows what each vest is worth after tax at this rate. Without a
rate, folio tax-rate prints the one that is set.
`,
		Example: `
  # Have vests taxed at 44.3%
  folio tax-rate 44.3%

  # Print the rate that is set
  folio tax-rate`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.printTaxRate()
			}

			rate, err := portfolio.ParseTaxRate(args[0])
			if err != nil {
				return err
			}

			return cmd.store.SetTaxRate(rate)
		},
	}
}

// printTaxRate prints the rate that vests are taxed at, or that there is none.
func (cmd *Folio) printTaxRate() error {
	rate, err := cmd.store.TaxRate()
	switch {
	case err != nil:
		return err
	case !rate.Valid:
		cmd.Println("No tax rate is set.")
	default:
		cmd.Printf("%s%%\n", rate.Decimal)
	}

	return nil
}
