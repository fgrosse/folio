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
		name: "tax-rate",
		get: func(store *portfolio.SQLiteStore) (string, bool, error) {
			rate, err := store.TaxRate()
			if err != nil || !rate.Valid {
				return "", false, err
			}

			return rate.Decimal.String() + "%", true, nil
		},
		set: func(store *portfolio.SQLiteStore, value string) error {
			rate, err := portfolio.ParseTaxRate(value)
			if err != nil {
				return err
			}

			return store.SetTaxRate(rate)
		},
	},
}

// errUnset is what folio config fails with for a key that is not set. Main exits 1 with it and
// prints nothing, as git config does, so that a script tells an unset key from a value by the exit
// code alone and never reads a message as the value.
var errUnset = errors.New("the key is not set")

// A configKey is a value of the configuration of an account, by the name that folio config knows
// it by. The values are kept in the store, each in a form of its own, so every key says how to read
// its value from there as text and how to store it from text.
type configKey struct {
	name string

	// get returns the value of the key as it is printed, and false if it is not set.
	get func(store *portfolio.SQLiteStore) (string, bool, error)

	// set parses value and stores it as the value of the key.
	set func(store *portfolio.SQLiteStore, value string) error
}

// ConfigCmd returns the "folio config" command.
func (cmd *Folio) ConfigCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "config [key] [value]",
		Short: "Get and set the configuration of your account",
		Long: `
Get and set the configuration of your account, the way git config does: with a key
and a value it sets the key to that value, and with a key alone it prints the value
the key is set to, or nothing and exits 1 if it is not set. Without a key it prints
every key that is set, as YAML, or as JSON with --output=json. The configuration
is kept in the database of the account.

The keys are:

  tax-rate   The rate that the shares still to vest are taxed at, as a percentage.
             A vest is taxed as income when it vests, at a rate that depends on the
             rest of the year's income and on where you live, so folio does not work
             it out. It takes one rate for every vest instead: your estimate of the
             rate at the top of your income. The Vesting view shows what each vest
             is worth after tax at this rate.
`,
		Example: `
  # Have vests taxed at 44.3%
  folio config tax-rate 44.3%

  # Print the rate that is set
  folio config tax-rate

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

			return key.set(cmd.store, args[1])
		},
	}

	// --output and -o are what kubectl names the flag that picks the format of what it prints, and
	// there is room in them for formats other than these two.
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

// printConfig prints the value of key, and fails with errUnset if it is not set.
func (cmd *Folio) printConfig(key configKey) error {
	value, ok, err := key.get(cmd.store)
	switch {
	case err != nil:
		return err
	case !ok:
		return errUnset
	}

	cmd.Println(value)
	return nil
}

// printAllConfig prints every key of the configuration that is set, with its value as printConfig
// prints it, in the format output: as YAML, or as a JSON object if output is json.
func (cmd *Folio) printAllConfig(output string) error {
	config := make(map[string]any)
	for _, key := range configKeys {
		value, ok, err := key.get(cmd.store)
		if err != nil {
			return fmt.Errorf("get %s: %w", key.name, err)
		}
		if ok {
			config[key.name] = value
		}
	}

	if output == "json" {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(config)
	}

	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	_, err = cmd.OutOrStdout().Write(out)
	return err
}
