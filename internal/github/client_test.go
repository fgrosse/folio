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

// TestClient_LatestErrors covers the ways GitHub says no: with a message worth passing on, such as
// when one address asked too often, and with a response that is no release at all.
func TestClient_LatestErrors(t *testing.T) {
	cases := map[string]struct {
		status int
		body   string
		error  string
	}{
		"rate limited": {
			status: http.StatusForbidden,
			body:   `{"message": "API rate limit exceeded for 203.0.113.7.", "documentation_url": "https://docs.github.com/rest"}`,
			error:  "no latest release: API rate limit exceeded for 203.0.113.7.",
		},
		"an error that is no JSON": {
			status: http.StatusBadGateway,
			body:   "<html>Bad Gateway</html>",
			error:  "no latest release: answered with 502 Bad Gateway",
		},
		"a response that is no JSON": {
			status: http.StatusOK,
			body:   "<html>Welcome</html>",
			error:  "no latest release: decode response: invalid character '<' looking for beginning of value",
		},
		"a release without a tag": {
			status: http.StatusOK,
			body:   `{"id": 405151455}`,
			error:  "no latest release: the response names none",
		},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(c.status)
				_, _ = w.Write([]byte(c.body))
			}))
			defer server.Close()

			client := New()
			client.apiURL = server.URL

			_, err := client.Latest(t.Context())
			require.EqualError(t, err, c.error)
		})
	}
}
