package xkcd_test

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithClient(t *testing.T) {
	clientUsed := false
	c := getClient(
		t,
		func(_ *http.Request) (*http.Response, error) {
			clientUsed = true
			return sendValidPost(t)
		},
		nil,
	)

	p, err := c.GetLatest(context.Background())
	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, p, "expected non-nil post")
	assert.True(t, clientUsed, "expected given client to be used")
}

func TestWithLogger(t *testing.T) {
	loggerUser := false
	c := getClient(
		t,
		nil,
		func(_ context.Context, _ slog.Record) error {
			loggerUser = true
			return nil
		},
	)

	p, err := c.GetLatest(context.Background())
	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, p, "expected non-nil post")
	assert.True(t, loggerUser, "expected given logger to be used")
}
