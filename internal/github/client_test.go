package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_Latest covers asking GitHub which release of folio is the newest: its version is the
// name of the tag it was released from, as the API states it.
func TestClient_Latest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/repos/fgrosse/folio/releases/latest", r.URL.Path)
		// GitHub turns away a request to its API that does not say who is asking.
		assert.NotEmpty(t, r.UserAgent())

		_, _ = w.Write([]byte(`{"id": 405151455, "tag_name": "v1.2.0", "name": "v1.2.0", "draft": false}`))
	}))
	defer server.Close()

	client := New()
	client.apiURL = server.URL

	version, err := client.Latest(t.Context())
	require.NoError(t, err)

	assert.Equal(t, "v1.2.0", version)
}
