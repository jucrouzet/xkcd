package xkcd

import "net/http"

// HTTPClient is an interface for http clients.
type HTTPClient interface {
	// Do performs the HTTP request and returns the response and error.
	Do(*http.Request) (*http.Response, error)
}

func (c *Client) getClient(given ...HTTPClient) HTTPClient {
	if len(given) > 0 {
		return given[0]
	}
	return c.getter
}
