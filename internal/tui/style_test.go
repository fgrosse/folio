package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

// TestDefaultStyle_GoldTotal covers the one thing that stands out in gold: the total account
// value, which is the number folio is for. The title of a dialog and the selected tab keep the
// purple of the selection, so that the gold is not shared with anything. The badge next to the total
// is as dim as the rest of the header and has no fill, so that the total still comes first.
func TestDefaultStyle_GoldTotal(t *testing.T) {
	style := DefaultStyle()
	gold := lipgloss.Color("220")

	assert.Equal(t, gold, style.Total.GetForeground())
	assert.True(t, style.Total.GetBold())

	assert.NotEqual(t, gold, style.DialogTitle.GetForeground())
	assert.NotEqual(t, gold, style.TabSelected.GetForeground())

	assert.Equal(t, lipgloss.NoColor{}, style.Badge.GetBackground())
	assert.Equal(t, style.Hint.GetForeground(), style.Badge.GetForeground())
	assert.False(t, style.Badge.GetBold())
}
