package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestConfigCmd_Set covers setting a value of the configuration: "folio config tax-rate" with a
// percentage stores the rate that vests are taxed at for the whole account, under the name of the
// key and written the one way that folio prints it, however it was typed.
func TestConfigCmd_Set(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "tax-rate", "44.30")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	value, err := store.GetConfig("tax-rate")
	require.NoError(t, err)
	assert.Equal(t, "44.3%", value)
}

// TestConfigCmd_Get covers asking for a value: "folio config tax-rate" without one prints the rate
// that is set. A key that is not set is an error, which goes to stderr and exits 1, so that a
// script that reads stdout never takes the message for the value.
func TestConfigCmd_Get(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "tax-rate")
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)

	assert.Equal(t, 1, cmd.Main())
	assert.Empty(t, out.String())
	assert.Equal(t, "Error: tax-rate is not set\n", stderr.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config", "tax-rate"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "44.3%\n", out.String())
}

// TestConfigCmd_UnknownKey covers a key that the configuration does not have, whose value can be
// neither set nor read: folio config says which keys there are.
func TestConfigCmd_UnknownKey(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "config", "tax", "44.3%")
	assert.EqualError(t, cmd.Execute(), `"tax" is no key of the configuration: use tax-rate, gains-tax-rate, show-net-summary`)

	cmd, _ = NewTestingCmd(t, "config", "tax")
	assert.EqualError(t, cmd.Execute(), `"tax" is no key of the configuration: use tax-rate, gains-tax-rate, show-net-summary`)
}

// TestConfigCmd_List covers "folio config" without a key, which prints the whole configuration as
// YAML: every key that is set, by its name, with the value it would print on its own. An account
// that has none set has an empty configuration.
func TestConfigCmd_List(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "{}\n", out.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "tax-rate: 44.3%\n", out.String())
}

// TestConfigCmd_ListJSON covers "folio config --output=json", which prints the whole configuration
// as a JSON object for a script to read, with the same keys and values as the YAML. -o is short for
// --output.
func TestConfigCmd_ListJSON(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "--output=json")
	var out bytes.Buffer
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, "{}\n", out.String())

	set := New()
	set.SetArgs([]string{"--db", dbPath, "config", "tax-rate", "44.3"})
	require.NoError(t, set.Execute())

	out.Reset()
	cmd = New()
	cmd.SetArgs([]string{"--db", dbPath, "config", "-o", "json"})
	cmd.SetOut(&out)

	require.NoError(t, cmd.Execute())
	assert.Equal(t, `{"tax-rate":"44.3%"}`+"\n", out.String())
}

// TestConfigCmd_OutputWithKey covers --output with a key, which is refused rather than ignored: it
// is the whole configuration that is printed in a format, and a single value prints as itself.
func TestConfigCmd_OutputWithKey(t *testing.T) {
	cmd, _ := NewTestingCmd(t, "config", "--output=json", "tax-rate")
	assert.EqualError(t, cmd.Execute(), "--output is for the whole configuration: leave out the key")
}

// TestConfigCmd_Outputs covers the formats --output takes: yaml, which is what folio config prints
// without the flag, and json. Any other is refused with the ones there are, rather than printed as
// one of them.
func TestConfigCmd_Outputs(t *testing.T) {
	cases := map[string]struct {
		output   string
		expected string
		error    string
	}{
		"yaml":    {output: "yaml", expected: "tax-rate: 44.3%\n"},
		"json":    {output: "json", expected: `{"tax-rate":"44.3%"}` + "\n"},
		"unknown": {output: "xml", error: `"xml" is no output format: use yaml or json`},
	}

	for name, tt := range cases {
		t.Run(name, func(t *testing.T) {
			set, dbPath := NewTestingCmd(t, "config", "tax-rate", "44.3%")
			require.NoError(t, set.Execute())

			cmd := New()
			cmd.SetArgs([]string{"--db", dbPath, "config", "--output", tt.output})
			var out bytes.Buffer
			cmd.SetOut(&out)

			err := cmd.Execute()
			if tt.error != "" {
				assert.EqualError(t, err, tt.error)
				assert.Empty(t, out.String())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expected, out.String())
		})
	}
}

// TestConfigCmd_ShowNetSummary covers choosing whether the TUI shows the values in its header after
// tax: "folio config show-net-summary" with true or false stores it, written the one way folio
// prints it, and refuses anything else before it reaches the store.
func TestConfigCmd_ShowNetSummary(t *testing.T) {
	cmd, dbPath := NewTestingCmd(t, "config", "show-net-summary", "True")
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	value, err := store.GetConfig("show-net-summary")
	require.NoError(t, err)
	assert.Equal(t, "true", value)

	cmd, _ = NewTestingCmd(t, "config", "show-net-summary", "net")
	assert.EqualError(t, cmd.Execute(), `"net" is neither true nor false`)
}

// TestConfigCmd_Unset covers taking a value back: "folio config --unset tax-rate" leaves the key as
// it was before it was set, so that vests are no longer taxed at any rate. It needs the key and
// nothing else, since a value next to it would say to set and to unset at once.
func TestConfigCmd_Unset(t *testing.T) {
	set, dbPath := NewTestingCmd(t, "config", "tax-rate", "44.3%")
	require.NoError(t, set.Execute())

	cmd := New()
	cmd.SetArgs([]string{"--db", dbPath, "config", "--unset", "tax-rate"})
	require.NoError(t, cmd.Execute())

	store, err := portfolio.NewStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })

	_, err = store.GetConfig("tax-rate")
	assert.ErrorIs(t, err, portfolio.ErrNotSet)

	cmd, _ = NewTestingCmd(t, "config", "--unset")
	assert.EqualError(t, cmd.Execute(), "--unset takes the key to unset and nothing else")

	cmd, _ = NewTestingCmd(t, "config", "--unset", "tax-rate", "44.3%")
	assert.EqualError(t, cmd.Execute(), "--unset takes the key to unset and nothing else")

	cmd, _ = NewTestingCmd(t, "config", "--unset", "tax")
	assert.EqualError(t, cmd.Execute(), `"tax" is no key of the configuration: use tax-rate, gains-tax-rate, show-net-summary`)
}
