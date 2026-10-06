package portfolio

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigKeys covers what a key of the configuration says about itself to a front end that
// lists the keys: what it is for, the values to pick from if it only takes some, and the value that
// applies while it is not set, if there is one. A value to pick and the default have to be ones the
// key itself takes, written as it stores them.
func TestConfigKeys(t *testing.T) {
	cases := map[string]struct {
		choices      []string
		defaultValue string
	}{
		TaxRateKey:        {},
		GainsTaxRateKey:   {},
		ShowNetSummaryKey: {choices: []string{"false", "true"}, defaultValue: "false"},
	}

	require.Len(t, ConfigKeys, len(cases), "every key should have a case")

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			key, err := ConfigKeyNamed(name)
			require.NoError(t, err)

			assert.NotEmpty(t, key.Description)
			assert.Equal(t, c.choices, key.Choices)
			assert.Equal(t, c.defaultValue, key.Default)

			for _, choice := range key.Choices {
				stored, err := key.Parse(choice)
				require.NoError(t, err)
				assert.Equal(t, choice, stored)
			}
		})
	}
}
