package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIndent covers the one thing indent has to get right that a string replacement would not: the
// lines that are empty stay empty, rather than becoming lines of spaces that no terminal shows and
// every golden file does.
func TestIndent(t *testing.T) {
	cases := map[string]struct {
		in       string
		expected string
	}{
		"a single line":             {in: "PANW 12", expected: "  PANW 12"},
		"every line":                {in: "PANW 12\n • 2026-03-15", expected: "  PANW 12\n   • 2026-03-15"},
		"an empty line stays empty": {in: "PANW 12\n\nTotal: 12", expected: "  PANW 12\n\n  Total: 12"},
		"a trailing newline stays":  {in: "PANW 12\n", expected: "  PANW 12\n"},
		"nothing at all":            {in: "", expected: ""},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, c.expected, indent(c.in))
		})
	}
}
