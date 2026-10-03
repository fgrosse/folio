package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// testConfigKeys are the keys the tests of the dialog edit: the tax rate, which is typed, and the
// potential basis, which is picked. They are named here rather than taken as portfolio.ConfigKeys
// so that a key added to folio does not change what these tests see.
func testConfigKeys(t *testing.T) []portfolio.ConfigKey {
	t.Helper()

	taxRate, err := portfolio.ConfigKeyNamed(portfolio.TaxRateKey)
	require.NoError(t, err)
	potential, err := portfolio.ConfigKeyNamed(portfolio.PotentialBasisKey)
	require.NoError(t, err)

	return []portfolio.ConfigKey{taxRate, potential}
}

// dialogText is what the dialog shows, without its styling.
func dialogText(d *ConfigDialog) string {
	return ansi.Strip(d.Layer().GetContent())
}

// TestConfigDialog_Rows covers what the dialog lists: one row for each key of the configuration,
// with the value it is set to, and under the rows what the selected key is for, which is the first
// when the dialog opens. A key that is not set shows its default, or says so if it has none. A key
// that takes one of a few values shows its value between the arrows that change it.
func TestConfigDialog_Rows(t *testing.T) {
	values := map[string]string{"tax-rate": "44.3%", "potential": "net"}
	d := NewConfigDialog(testConfigKeys(t), values, dialogWidth, DefaultStyle())

	text := dialogText(d)
	assert.Contains(t, text, "Configuration")
	assert.Contains(t, text, "> tax-rate   44.3%")
	assert.Contains(t, text, "  potential  ‹ net ›")
	assert.Contains(t, text, "The rate that the shares still to vest are taxed at")
	assert.NotContains(t, text, "Which potential value", "only the selected key is described")

	d = NewConfigDialog(testConfigKeys(t), nil, dialogWidth, DefaultStyle())

	text = dialogText(d)
	assert.Contains(t, text, "> tax-rate   not set")
	assert.Contains(t, text, "  potential  ‹ gross ›")
}
