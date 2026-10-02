package portfolio

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// quoterFunc is a Quoter made of a function, for a test to answer with whatever it needs.
type quoterFunc func(symbol string) (Quote, error)

func (f quoterFunc) Quote(_ context.Context, symbol string) (Quote, error) {
	return f(symbol)
}

// TestFetchQuotes covers getting the prices of an account: the quote of every symbol asked for, by
// symbol. A symbol that has none does not cost the others theirs: the quotes that could be read
// come back along with an error that says which could not, so that an account with one mistyped
// symbol still shows what the rest of it is worth.
func TestFetchQuotes(t *testing.T) {
	panw := Quote{Symbol: "PANW", Price: shares("396.25")}
	aapl := Quote{Symbol: "AAPL", Price: shares("330.32")}
	quoter := quoterFunc(func(symbol string) (Quote, error) {
		switch symbol {
		case "PANW":
			return panw, nil
		case "AAPL":
			return aapl, nil
		default:
			return Quote{}, errors.New("no quote of " + symbol)
		}
	})

	quotes, err := FetchQuotes(t.Context(), quoter, []string{"AAPL", "PANW"})
	assert.NoError(t, err)
	assert.Equal(t, map[string]Quote{"AAPL": aapl, "PANW": panw}, quotes)

	quotes, err = FetchQuotes(t.Context(), quoter, []string{"AAPL", "NOPE", "PANW", "NADA"})
	assert.EqualError(t, err, "no quote of NOPE\nno quote of NADA")
	assert.Equal(t, map[string]Quote{"AAPL": aapl, "PANW": panw}, quotes)
}
