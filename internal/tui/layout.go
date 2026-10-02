package tui

import (
	"strings"
)

// viewIndent is how far a view's content sits from the left edge of the frame. It clears the
// table's border and the padding of its first cell, so that a column of text lines up with the
// first column of the table and switching tabs leaves the text where it was.
const viewIndent = "  "

// indent puts every line of s behind viewIndent, except the empty ones. A line of nothing but
// spaces reads the same as an empty one on a terminal, but not in a golden file, and not to a
// terminal that draws the selection or a background color across the padding.
func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = viewIndent + line
		}
	}

	return strings.Join(lines, "\n")
}
