package yahoo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fgrosse/folio/internal/portfolio"
)

// TestClient_Quote covers reading a quote off Yahoo's chart endpoint: the price and the previous
// close exactly as the response writes them, the currency, and the time of the trade the price is
// from rather than the time of the request, so that a quote fetched on a Sunday says it is Friday's.
func TestClient_Quote(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v8/finance/chart/PANW", r.URL.Path)
		// Yahoo turns away a request that does not look like it comes from a browser.
		assert.Contains(t, r.UserAgent(), "Mozilla")

		response, err := os.ReadFile("testdata/chart_PANW.json")
		assert.NoError(t, err)
		_, _ = w.Write(response)
	}))
	defer server.Close()

	client := New()
	client.baseURL = server.URL

	quote, err := client.Quote(t.Context(), "PANW")
	require.NoError(t, err)

	expected := portfolio.Quote{
		Symbol:        "PANW",
		Price:         decimal.RequireFromString("396.25"),
		PreviousClose: decimal.RequireFromString("397.31"),
		Currency:      "USD",
		At:            time.Unix(1790884800, 0).UTC(),
	}
	assert.Equal(t, expected, quote)
}
