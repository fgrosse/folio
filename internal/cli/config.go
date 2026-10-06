package cli

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/fgrosse/folio/internal/portfolio"
)

// ConfigCmd returns the "folio config" command.
func (cmd *Folio) ConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get and set the configuration of your account",
		Long: `
Get and set the configuration of your account: with a key and a value it sets the
key to that value, and with a key alone it prints the value the key is set to.
Without a key it prints every key that is set, as YAML, or as JSON with
--output=json. With --unset and a key it takes the value of the key back. The
configuration is kept in the database of the account.

The keys are:

  tax-rate
      The rate that the shares still to vest are taxed at, as a percentage. A
      vest is taxed as income when it vests, at a rate that depends on the rest
      of the year's income and on where you live, so folio does not work it
      out. It takes one rate for every vest instead: your estimate of the rate
      at the top of your income. The Vesting view shows what each vest is worth
      after tax at this rate.

  gains-tax-rate
      The rate that the gain of a sale is taxed at, as a percentage: what
      selling the shares you hold would cost of what they gained since you got
      them. With it, the Holdings view shows the tax that selling each lot at
      the price of today would cost. A lot that lost is not taxed.

  show-net-summary
      Whether the TUI shows the values in its header after tax, true or false.
      With true, the potential value is what is left of the shares still to
      vest after tax at the tax rate, and the current value what is left of the
      shares you hold after tax on the gains at the gains tax rate, as if you
      sold them all today. The total is the two added up. A value without its
      rate stays as the bank states it. It is false if it is not set, which
      shows all values as the bank states them.
`,
		Example: `
  # Have vests taxed at 44.3%
  folio config tax-rate 44.3%

  # Print the rate that is set
  folio config tax-rate

  # Have the gain of a sale taxed at 26.4%
  folio config gains-tax-rate 26.4%

  # Show the values after tax in the header of the TUI
  folio config show-net-summary true

  # Have vests taxed at no rate again
  folio config --unset tax-rate

  # Print the whole configuration
  folio config

  # Print the whole configuration for a script
  folio config --output=json`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			output, err := c.Flags().GetString("output")
			if err != nil {
				return err
			}

			unset, err := c.Flags().GetBool("unset")
			switch {
			case err != nil:
				return err
			case unset:
				return cmd.unsetConfig(args)
			case len(args) == 0:
				return cmd.printAllConfig(output)
			case c.Flags().Changed("output"):
				return errors.New("--output is for the whole configuration: leave out the key")
			}

			key, err := portfolio.ConfigKeyNamed(args[0])
			if err != nil {
				return err
			}

			if len(args) == 1 {
				return cmd.printConfig(key)
			}

			value, err := key.Parse(args[1])
			if err != nil {
				return err
			}

			return cmd.store.SetConfig(key.Name, value)
		},
	}

	// One flag that names the format, rather than a flag for each, has room for formats other than
	// these two.
	c.Flags().StringP("output", "o", "yaml", "print the whole configuration as yaml or json")
	c.Flags().Bool("unset", false, "take the value of the key back, which leaves it not set")

	return c
}

// unsetConfig takes back the value of the key that args names, which have to be that key alone.
func (cmd *Folio) unsetConfig(args []string) error {
	if len(args) != 1 {
		return errors.New("--unset takes the key to unset and nothing else")
	}

	key, err := portfolio.ConfigKeyNamed(args[0])
	if err != nil {
		return err
	}

	return cmd.store.UnsetConfig(key.Name)
}

// printConfig prints the value of key, and fails with portfolio.ErrNotSet if it is not set.
func (cmd *Folio) printConfig(key portfolio.ConfigKey) error {
	value, err := cmd.store.GetConfig(key.Name)
	if err != nil {
		return err
	}

	cmd.Println(value)
	return nil
}

// printAllConfig prints every key of the configuration that is set, with its value as printConfig
// prints it, in the format output, which is yaml or json.
func (cmd *Folio) printAllConfig(output string) error {
	config := make(map[string]any)
	for _, key := range portfolio.ConfigKeys {
		value, err := cmd.store.GetConfig(key.Name)
		switch {
		case errors.Is(err, portfolio.ErrNotSet):
			continue
		case err != nil:
			return fmt.Errorf("get %s: %w", key.Name, err)
		}

		config[key.Name] = value
	}

	switch output {
	case "yaml":
		out, err := yaml.Marshal(config)
		if err != nil {
			return err
		}

		_, err = cmd.OutOrStdout().Write(out)
		return err
	case "json":
		return json.NewEncoder(cmd.OutOrStdout()).Encode(config)
	default:
		return fmt.Errorf("%q is no output format: use yaml or json", output)
	}
}
