package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"github.com/fgrosse/folio/internal/portfolio"
)

// configKeys are the keys of the configuration, in the order they are listed in.
var configKeys = []configKey{
	{
		name: portfolio.TaxRateKey,
		parse: func(value string) (string, error) {
			rate, err := portfolio.ParseTaxRate(value)
			if err != nil {
				return "", err
			}

			return rate.String() + "%", nil
		},
	},
	{
		name: portfolio.PotentialBasisKey,
		parse: func(value string) (string, error) {
			basis, err := portfolio.ParsePotentialBasis(value)
			return string(basis), err
		},
	},
}

// A configKey is a value of the configuration of an account, by the name that folio config knows
// it by and the store keeps it under. The store keeps any text it is given, so it is up to the key
// to refuse a value that is not one, before it is stored and read by a part of folio that does not
// expect it.
type configKey struct {
	name string

	// parse checks value and returns it written the one way the store keeps it and folio config
	// prints it, or an error that says what is wrong with it.
	parse func(value string) (string, error)
}

// ConfigCmd returns the "folio config" command.
func (cmd *Folio) ConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get and set the configuration of your account",
		Long: `
Get and set the configuration of your account: with a key and a value it sets the
key to that value, and with a key alone it prints the value the key is set to.
Without a key it prints every key that is set, as YAML, or as JSON with
--output=json. The configuration is kept in the database of the account.

The keys are:

  tax-rate   The rate that the shares still to vest are taxed at, as a percentage.
             A vest is taxed as income when it vests, at a rate that depends on the
             rest of the year's income and on where you live, so folio does not work
             it out. It takes one rate for every vest instead: your estimate of the
             rate at the top of your income. The Vesting view shows what each vest
             is worth after tax at this rate.

  potential  Which potential value the TUI shows in its header: gross, the value
             of the shares still to vest as the bank states it, or net, what is
             left of it after tax at the tax rate. With net, the total is the
             current value and the potential value after tax. It is gross if it
             is not set, and gross too without a tax rate to take off.
`,
		Example: `
  # Have vests taxed at 44.3%
  folio config tax-rate 44.3%

  # Print the rate that is set
  folio config tax-rate

  # Show the potential value after tax in the TUI
  folio config potential net

  # Print the whole configuration
  folio config

  # Print the whole configuration for a script
  folio config --output=json`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(c *cobra.Command, args []string) error {
			output, err := c.Flags().GetString("output")
			switch {
			case err != nil:
				return err
			case len(args) == 0:
				return cmd.printAllConfig(output)
			case c.Flags().Changed("output"):
				return errors.New("--output is for the whole configuration: leave out the key")
			}

			key, err := configKeyNamed(args[0])
			if err != nil {
				return err
			}

			if len(args) == 1 {
				return cmd.printConfig(key)
			}

			value, err := key.parse(args[1])
			if err != nil {
				return err
			}

			return cmd.store.SetConfig(key.name, value)
		},
	}

	// One flag that names the format, rather than a flag for each, has room for formats other than
	// these two.
	c.Flags().StringP("output", "o", "yaml", "print the whole configuration as yaml or json")

	return c
}

// configKeyNamed returns the key of the configuration with the given name.
func configKeyNamed(name string) (configKey, error) {
	names := make([]string, len(configKeys))
	for i, key := range configKeys {
		if key.name == name {
			return key, nil
		}
		names[i] = key.name
	}

	return configKey{}, fmt.Errorf("%q is no key of the configuration: use %s", name, strings.Join(names, ", "))
}

// printConfig prints the value of key, and fails with portfolio.ErrNotSet if it is not set.
func (cmd *Folio) printConfig(key configKey) error {
	value, err := cmd.store.GetConfig(key.name)
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
	for _, key := range configKeys {
		value, err := cmd.store.GetConfig(key.name)
		switch {
		case errors.Is(err, portfolio.ErrNotSet):
			continue
		case err != nil:
			return fmt.Errorf("get %s: %w", key.name, err)
		}

		config[key.name] = value
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
