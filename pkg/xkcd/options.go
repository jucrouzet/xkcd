package xkcd

import "log/slog"

// ClientOption is a function that configures a Client.
type ClientOption func(c *Client)

// WithClient sets the http client for http operations.
func WithClient(g Getter) ClientOption {
	return func(c *Client) {
		c.getter = g
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}
