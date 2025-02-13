package xkcd

import "net/http"

// Getter is an interface for making HTTP requests.
type Getter interface {
	// Do performs the HTTP request and returns the response and error.
	Do(*http.Request) (*http.Response, error)
}

func (c *Client) getClient(given ...Getter) Getter {
	if len(given) > 0 {
		return given[0]
	}
	return c.getter
}
