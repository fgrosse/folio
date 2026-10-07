// Package github reads the releases of folio from GitHub, where they are published.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// repository is where folio is developed and released.
const repository = "fgrosse/folio"

// defaultAPIURL is where GitHub serves its API. Reading the releases of a public repository needs
// no token, and is limited to 60 requests an hour from one address, which is plenty for an update.
const defaultAPIURL = "https://api.github.com"

// userAgent is what the client introduces itself as. GitHub turns away a request to its API that
// has none.
const userAgent = "folio"

// A Client reads the releases of folio from GitHub. It is the update.Source of the program.
type Client struct {
	http   *http.Client
	apiURL string
}

// New returns a Client that gives up on a request after two minutes. That is long for a request,
// and is what the archive of a release may take over a slow connection.
func New() *Client {
	return &Client{
		http:   &http.Client{Timeout: 2 * time.Minute},
		apiURL: defaultAPIURL,
	}
}

// Latest returns the version of the newest release, which is the name of its tag, such as
// "v1.2.0". GitHub leaves drafts and prereleases out of what it calls the latest.
func (c *Client) Latest(ctx context.Context) (string, error) {
	version, err := c.latest(ctx)
	if err != nil {
		return "", fmt.Errorf("no latest release: %w", err)
	}

	return version, nil
}

func (c *Client) latest(ctx context.Context) (string, error) {
	endpoint := c.apiURL + "/repos/" + repository + "/releases/latest"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	// The body is read before the status is looked at: GitHub says why it turned a request away
	// in a message, which says more than the status does. A body that is no JSON leaves the status.
	var release struct {
		TagName string `json:"tag_name"`
		Message string `json:"message"`
	}
	err = json.NewDecoder(resp.Body).Decode(&release)
	switch {
	case err == nil && resp.StatusCode != http.StatusOK && release.Message != "":
		return "", errors.New(release.Message)
	case resp.StatusCode != http.StatusOK:
		return "", fmt.Errorf("answered with %s", resp.Status)
	case err != nil:
		return "", fmt.Errorf("decode response: %w", err)
	case release.TagName == "":
		return "", errors.New("the response names none")
	}

	return release.TagName, nil
}
