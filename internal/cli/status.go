package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// StatusCmd returns the "folio status" command.
func (cmd *Folio) StatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show what the account is worth right now",
		Long: `
Show the three values of the account at the latest prices:

  Current    what the shares you hold would sell for
  Potential  what the shares that are still to vest, or vested and not released, are worth
  Total      the two added up
`,
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			lots, err := cmd.store.Lots()
			if err != nil {
				return err
			}

			grants, err := cmd.store.Grants()
			if err != nil {
				return err
			}

			quotes, err := portfolio.FetchQuotes(c.Context(), cmd.quoter, portfolio.Symbols(lots, grants))
			if err != nil {
				return err
			}

			for _, quote := range quotes {
				if err := cmd.store.SaveQuote(quote); err != nil {
					return fmt.Errorf("save quote of %s: %w", quote.Symbol, err)
				}
			}

			cmd.printAccount(portfolio.NewAccount(lots, grants, quotes))
			return nil
		},
	}
}

// printAccount writes the three values of account, one to a line, the amounts lined up on the right
// the way a column of figures is.
func (cmd *Folio) printAccount(account portfolio.Account) {
	values := []struct {
		label  string
		amount string
	}{
		{"Current", portfolio.FormatUSD(account.Current)},
		{"Potential", portfolio.FormatUSD(account.Potential)},
		{"Total", portfolio.FormatUSD(account.Total())},
	}

	var width int
	for _, value := range values {
		width = max(width, len(value.amount))
	}

	for _, value := range values {
		cmd.Printf("%-9s  %*s\n", value.label, width, value.amount)
	}
}
