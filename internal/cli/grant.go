package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/fgrosse/folio/internal/portfolio"
)

// GrantCmd returns the "folio grant" command.
func (cmd *Folio) GrantCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "grant <name>: <shares> <symbol> <interval> x<count>|<percent>/... from <YYYY-MM-DD>",
		Short: "Record a grant of shares that vest over time",
		Long: `
Record a grant: an award of shares that vest over time, such as a grant of RSUs. Its vests
are what the potential account value counts, until each is released into a lot.

The grant is given as its name, a colon, and its schedule. The interval is monthly,
quarterly or yearly, and the day after "from" is that of the first vest.

With a count such as x24, the grant vests that many shares that many times. With
percentages such as 10/20/30/40, the shares are those of the whole grant, and each
percentage is the part of them that vests in one year, spread evenly over the vests of that
year. Only whole shares vest then, and a fraction left over arrives with a later vest.

Quoting is optional: any positional arguments are joined with spaces.
`,
		Example: `
  # 10 shares on the 15th of every month, 24 times
  folio grant "Acquisition payout: 10 PANW monthly x24 from 2026-01-15"

  # 400 shares over four years, every quarter, 10% of them in the first year and 40% in the last
  folio grant "RSU 2025: 400 PANW quarterly 10/20/30/40 from 2026-02-20"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			grant, err := portfolio.NewGrant(strings.Join(args, " "))
			if err != nil {
				return err
			}

			return cmd.store.SaveGrant(grant)
		},
	}
}
