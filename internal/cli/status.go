package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// StatusCmd returns the "folio status" command.
func (cmd *Folio) StatusCmd() *cobra.Command {
	c := &cobra.Command{
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
			asJSON, err := c.Flags().GetBool("json")
			if err != nil {
				return err
			}

			lots, err := cmd.store.Lots()
			if err != nil {
				return err
			}

			grants, err := cmd.store.Grants()
			if err != nil {
				return err
			}

			quotes, err := cmd.quotes(c.Context(), portfolio.Symbols(lots, grants))
			if err != nil {
				return err
			}

			account := portfolio.NewAccount(lots, grants, quotes)
			if asJSON {
				return cmd.printAccountJSON(account)
			}

			cmd.printAccount(account)
			return nil
		},
	}

	c.Flags().Bool("json", false, "print the values as a JSON object")

	return c
}

// quotes returns the latest quote of each of symbols: fetched just now where that worked, and saved
// to the database for the next run, and otherwise the one an earlier run left there. Why a quote
// could not be fetched goes to stderr as a warning, which leaves stdout to the values.
func (cmd *Folio) quotes(ctx context.Context, symbols []string) (map[string]portfolio.Quote, error) {
	quotes, err := cmd.store.Quotes()
	if err != nil {
		return nil, err
	}

	fetched, err := portfolio.FetchQuotes(ctx, cmd.quoter, symbols)
	if err != nil {
		for line := range strings.SplitSeq(err.Error(), "\n") {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warning:", line)
		}
	}

	for symbol, quote := range fetched {
		if err := cmd.store.SaveQuote(quote); err != nil {
			return nil, fmt.Errorf("save quote of %s: %w", symbol, err)
		}

		quotes[symbol] = quote
	}

	return quotes, nil
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

// printAccountJSON writes the values of account as one JSON object, the amounts as decimals to the
// cent in strings, which no reader will take for floats.
func (cmd *Folio) printAccountJSON(account portfolio.Account) error {
	unpriced := account.Unpriced
	if unpriced == nil {
		unpriced = []string{} // an empty list rather than null
	}

	return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
		Current   string   `json:"current"`
		Potential string   `json:"potential"`
		Total     string   `json:"total"`
		Currency  string   `json:"currency"`
		Unpriced  []string `json:"unpriced"`
	}{
		Current:   account.Current.StringFixed(2),
		Potential: account.Potential.StringFixed(2),
		Total:     account.Total().StringFixed(2),
		Currency:  "USD",
		Unpriced:  unpriced,
	})
}
