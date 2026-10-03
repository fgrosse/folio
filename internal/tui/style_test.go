package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
)

// TestDefaultStyle_Gold covers the two things that stand out in gold and bold: the total account
// value, which is the number folio is for, and the title of a dialog, which says what is being
// asked. The selected tab keeps the purple of the selection.
func TestDefaultStyle_Gold(t *testing.T) {
	style := DefaultStyle()
	gold := lipgloss.Color("220")

	assert.Equal(t, gold, style.Total.GetForeground())
	assert.True(t, style.Total.GetBold())

	assert.Equal(t, gold, style.DialogTitle.GetForeground())
	assert.True(t, style.DialogTitle.GetBold())

	assert.NotEqual(t, gold, style.TabSelected.GetForeground())
}
