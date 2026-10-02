package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// ReleaseCmd returns the "folio release" command.
func (cmd *Folio) ReleaseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "release <YYYY-MM-DD> <shares> [@<cost>]",
		Short: "Release a vest into shares that you hold",
		Long: `
Release the vest of the given day: turn it into a lot of the shares that actually arrived,
which are fewer than vested when some were sold or withheld for tax. From then on they
count towards the current account value instead of the potential one.

The cost, after an "@", is what one share was worth on the day of the vest. It is what the
lot cost, and what a later gain is measured from.
`,
		Example: `
  # Of the shares that vested on the 1st of August, 250 arrived, worth $162.50 each that day
  folio release 2025-08-01 250 @162.50`,
		Args: cobra.RangeArgs(2, 3),
		RunE: func(c *cobra.Command, args []string) error {
			date, err := portfolio.ParseDay(args[0])
			if err != nil {
				return err
			}

			shares, err := decimal.NewFromString(args[1])
			if err != nil {
				return fmt.Errorf("%q is not a number of shares", args[1])
			}

			cost := decimal.Zero
			if len(args) == 3 {
				cost, err = portfolio.ParseCost(args[2])
				if err != nil {
					return err
				}
			}

			id, err := cmd.vestOn(date)
			if err != nil {
				return err
			}

			return cmd.store.ReleaseVest(id, shares, cost)
		},
	}
}

// vestOn returns the ID of the vest of the given day that has not been released.
func (cmd *Folio) vestOn(date time.Time) (int, error) {
	grants, err := cmd.store.Grants()
	if err != nil {
		return 0, err
	}

	for _, grant := range grants {
		for _, vest := range grant.Vests {
			if vest.Date.Equal(date) && !vest.Released {
				return vest.ID, nil
			}
		}
	}

	return 0, errors.New("no vest of that day is waiting to be released")
}
