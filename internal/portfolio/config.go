package portfolio

import (
	"fmt"
	"strings"
)

// ConfigKeys are the keys of the configuration, in the order they are listed in. They are here
// rather than with one of the front ends so that folio config and the TUI offer the same keys and
// refuse the same values.
var ConfigKeys = []ConfigKey{
	{
		Name: TaxRateKey,
		Description: "The rate that the shares still to vest are taxed at, as a percentage: " +
			"your estimate of the rate at the top of your income.",
		Parse: parseStoredTaxRate,
	},
	{
		Name: PotentialBasisKey,
		Description: "Which potential value the header shows: gross, as the bank states it, " +
			"or net, what is left of it after tax at the tax rate.",
		Choices: []string{string(Gross), string(Net)},
		Default: string(Gross),
		Parse:   parseStoredBasis,
	},
	{
		Name: GainsTaxRateKey,
		Description: "The rate that the gain of a sale is taxed at, as a percentage: what " +
			"selling the shares you hold would cost of what they gained since you got them.",
		Parse: parseStoredTaxRate,
	},
	{
		Name: CurrentBasisKey,
		Description: "Which current value the header shows: gross, as the bank states it, " +
			"or net, what is left of it after tax on the gains at the gains tax rate.",
		Choices: []string{string(Gross), string(Net)},
		Default: string(Gross),
		Parse:   parseStoredBasis,
	},
}

// parseStoredTaxRate checks a tax rate and returns it as the store keeps it, with its percent sign.
func parseStoredTaxRate(value string) (string, error) {
	rate, err := ParseTaxRate(value)
	if err != nil {
		return "", err
	}

	return rate.String() + "%", nil
}

// parseStoredBasis checks a basis and returns it as the store keeps it, in small letters.
func parseStoredBasis(value string) (string, error) {
	basis, err := ParseBasis(value)
	return string(basis), err
}

// A ConfigKey is a value of the configuration of an account, by the name that folio config knows
// it by and the store keeps it under. The store keeps any text it is given, so it is up to the key
// to refuse a value that is not one, before it is stored and read by a part of folio that does not
// expect it.
type ConfigKey struct {
	Name string

	// Description says what the key is for, in a sentence, to whoever is about to set it.
	Description string

	// Choices are the values the key takes, as Parse returns them, if it takes only some. A front
	// end can offer these to pick from rather than a field to type into.
	Choices []string

	// Default is the value that applies while the key is not set, as Parse returns it. It is empty
	// for a key that has none, such as the tax rate, where not set means that there is no rate.
	Default string

	// Parse checks value and returns it written the one way the store keeps it and folio prints
	// it, or an error that says what is wrong with it.
	Parse func(value string) (string, error)
}

// ConfigKeyNamed returns the key of the configuration with the given name.
func ConfigKeyNamed(name string) (ConfigKey, error) {
	names := make([]string, len(ConfigKeys))
	for i, key := range ConfigKeys {
		if key.Name == name {
			return key, nil
		}
		names[i] = key.Name
	}

	return ConfigKey{}, fmt.Errorf("%q is no key of the configuration: use %s", name, strings.Join(names, ", "))
}
