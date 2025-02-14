// Package xkcd provides functions to interact with the xkcd API.
package xkcd

import (
	"log/slog"
	"net/http"
)

// Client is a xkcd api client.
type Client struct {
	defaultClient HTTPClient
	logger        *slog.Logger
}

// New returns a new xkcd API client with the provided options.
func New(opts ...ClientOption) *Client {
	client := &Client{
		defaultClient: &http.Client{},
		logger:        slog.New(slog.NewTextHandler(nullWriter{}, nil)),
	}
	for _, opt := range opts {
		opt(client)
	}
	return client
}

type nullWriter struct{}

// Write implements io.Writer for the null logger.
func (nullWriter) Write([]byte) (int, error) { return 0, nil }
