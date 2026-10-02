package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// ReleaseCmd returns the "folio release" command.
func (cmd *Folio) ReleaseCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "release <YYYY-MM-DD> <shares> [@<cost>]",
		Short: "Release a vest into shares that you hold",
		Long: `
Release the vest of the given day: turn it into a lot of the shares that actually arrived,
which are fewer than vested when some were sold or withheld for tax. From then on they
count towards the current account value instead of the potential one.

The cost, after an "@", is what one share was worth on the day of the vest. It is what the
lot cost, and what a later gain is measured from.

If more than one grant vests on that day, --grant says which one is meant.
`,
		Example: `
  # Of the shares that vested on the 1st of August, 250 arrived, worth $162.50 each that day
  folio release 2025-08-01 250 @162.50`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			date, err := portfolio.ParseDay(args[0])
			if err != nil {
				return err
			}

			// The shares and the cost are one spec, however the shell split them into arguments.
			shares, cost, err := portfolio.ParseRelease(strings.Join(args[1:], " "))
			if err != nil {
				return err
			}

			grant, err := c.Flags().GetString("grant")
			if err != nil {
				return err
			}

			id, err := cmd.vestOn(date, grant)
			if err != nil {
				return err
			}

			return cmd.store.ReleaseVest(id, shares, cost)
		},
	}

	c.Flags().String("grant", "", "the name of the grant, if more than one vests on that day")

	return c
}

// vestOn returns the ID of the vest of the given day that has not been released: the one of the
// grant with the given name, or, without a name, the only one there is. Two grants that vest on the
// same day leave it open which is meant, and need the name.
func (cmd *Folio) vestOn(date time.Time, grantName string) (int, error) {
	grants, err := cmd.store.Grants()
	if err != nil {
		return 0, err
	}

	var (
		ids   []int
		names []string
	)
	for _, grant := range grants {
		if grantName != "" && grant.Name != grantName {
			continue
		}

		for _, vest := range grant.Vests {
			if vest.Date.Equal(date) && !vest.Released {
				ids = append(ids, vest.ID)
				names = append(names, strconv.Quote(grant.Name))
			}
		}
	}

	on := date.Format(time.DateOnly)
	switch {
	case len(ids) == 1:
		return ids[0], nil
	case len(ids) > 1:
		return 0, fmt.Errorf("%d grants vest on %s: name one with --grant, %s", len(ids), on, strings.Join(names, " or "))
	case grantName != "":
		return 0, fmt.Errorf("no vest of %q on %s is waiting to be released", grantName, on)
	default:
		return 0, fmt.Errorf("no vest of %s is waiting to be released", on)
	}
}
