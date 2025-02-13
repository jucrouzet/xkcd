package xkcd_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_getClient(t *testing.T) {
	clientUsed := false
	c := getClient(
		t,
		func(_ *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("this client should not be used")
		},
		nil,
	)
	specificClient := &http.Client{Transport: &mockRoundTripper{
		mock: func(_ *http.Request) (*http.Response, error) {
			clientUsed = true
			return sendValidPost(t)
		},
		t: t,
	}}

	p, err := c.GetLatest(context.Background(), specificClient)
	assert.NoError(t, err, "expected no error")
	assert.NotNil(t, p, "expected non-nil post")
	assert.True(t, clientUsed, "expected given client to be used")
}
