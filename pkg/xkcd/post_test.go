package xkcd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

func TestClient_GetPost(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		//nolint:revive,staticcheck // It's safe here to use string key
		ctx := context.WithValue(context.Background(), "test-key", "test-value")
		expected, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			require.Equal(t, ctx, r.Context(), "given context should used for request")
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, p, "expected non-nil post")
		if !cmp.Equal(expected, p, cmpopts.IgnoreFields(xkcd.Post{}, "defaultClient", "logger")) {
			t.Errorf("expected post to be correctly parsed: %s", cmp.Diff(expected, p, cmpopts.IgnoreFields(xkcd.Post{}, "defaultClient", "logger")))
		}
	})

	t.Run("invalid number", func(t *testing.T) {
		c := getClient(t, nil, nil)
		p, err := c.GetPost(context.Background(), -1)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrNoSuchPost)
		p, err = c.GetPost(context.Background(), 0)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrNoSuchPost, "expected ErrNoSuchPost")
	})

	t.Run("request failed", func(t *testing.T) {
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New("kaboom")
		}, nil)
		p, err := c.GetPost(context.Background(), 1)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected ErrAPIError")
		assert.ErrorContains(t, err, "kaboom", "expected error to wrap original error")
	})

	t.Run("reading failed", func(t *testing.T) {
		_, resp := getRandomPost(t, errors.New("i cannot read"), nil)
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return resp, nil
		}, nil)
		p, err := c.GetPost(context.Background(), 1)
		assert.Nil(t, p, "expected nil post")
		assert.Error(t, err, "expected an error")
		assert.ErrorContains(t, err, "i cannot read", "expected error to wrap original error")
	})

	t.Run("log if closing body failed, but no error returned (defer)", func(t *testing.T) {
		closeLogged := false
		expected, resp := getRandomPost(
			t,
			nil,
			errors.New("kaboom"),
		)
		c := getClient(
			t,
			func(_ *http.Request) (*http.Response, error) {
				return resp, nil
			},
			func(_ context.Context, record slog.Record) error {
				if record.Message == "failed to close response body" {
					closeLogged = true
				}
				return nil
			},
		)
		p, err := c.GetPost(context.Background(), 1)
		assert.NoError(t, err, "expected no error")
		assert.NotNil(t, p, "expected non-nil post")
		if !cmp.Equal(expected, p, cmpopts.IgnoreFields(xkcd.Post{}, "defaultClient", "logger")) {
			t.Errorf("expected post to be correctly parsed: %s", cmp.Diff(expected, p, cmpopts.IgnoreFields(xkcd.Post{}, "defaultClient", "logger")))
		}
		assert.True(t, closeLogged, "expected logger to be called for a close error")
	})

	t.Run("no such post", func(t *testing.T) {
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Not found",
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader("404 Not Found")),
			}, nil
		}, nil)
		p, err := c.GetPost(context.Background(), 1)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrNoSuchPost, "expected ErrNoSuchPost")
	})

	t.Run("non 200 or 404 status", func(t *testing.T) {
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Internal server error",
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("I'm sick")),
			}, nil
		}, nil)
		p, err := c.GetPost(context.Background(), 1)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected ErrAPIError")
		assert.ErrorContains(t, err, "503", "expected status code to be included in error message")
	})

	t.Run("invalid json", func(t *testing.T) {
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				Status:     "Ok",
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("<xml>Invalid JSON</xml>")),
			}, nil
		}, nil)
		p, err := c.GetPost(context.Background(), 1)
		assert.Error(t, err, "expected an error")
		assert.Nil(t, p, "expected nil post")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected ErrAPIError")
		assert.ErrorContains(t, err, "failed to decode response", "expected error to be explicit about invalid JSON")
		assert.ErrorContains(t, err, "invalid character '<'", "expected error to wrap original error")
	})

	type postCase struct {
		name          string
		preparePost   func(*xkcd.Post)
		errorContains string
	}
	cases := []postCase{
		{
			name:          "empty struct",
			preparePost:   func(post *xkcd.Post) { *post = xkcd.Post{} },
			errorContains: "post number is zero",
		},
		{
			name: "num is zero",
			preparePost: func(post *xkcd.Post) {
				post.Num = 0
			},
			errorContains: "post number is zero",
		},
		{
			name: "invalid day",
			preparePost: func(post *xkcd.Post) {
				post.Day = "32"
			},
			errorContains: "failed to parse date day: invalid value for day: 32",
		},
		{
			name: "invalid month",
			preparePost: func(post *xkcd.Post) {
				post.Month = ""
			},
			errorContains: "failed to parse date month: strconv.Atoi: parsing \"\": invalid syntax",
		},
		{
			name: "invalid month",
			preparePost: func(post *xkcd.Post) {
				post.Month = "13"
			},
			errorContains: "failed to parse date month: invalid value for month: 13",
		},
		{
			name: "invalid year",
			preparePost: func(post *xkcd.Post) {
				post.Year = "hello"
			},
			errorContains: "failed to parse date year: strconv.Atoi: parsing \"hello\": invalid syntax",
		},
		{
			name: "empty image",
			preparePost: func(post *xkcd.Post) {
				post.Img = ""
			},
			errorContains: "post image URL is invalid: URL is empty",
		},
		{
			name: "invalid image",
			preparePost: func(post *xkcd.Post) {
				post.Img = "http ://lol.com/a"
			},
			errorContains: "post image URL is invalid: invalid syntax",
		},
		{
			name: "invalid image url scheme",
			preparePost: func(post *xkcd.Post) {
				post.Img = "mailto:contact@juliencrouzet.fr"
			},
			errorContains: "post image URL is invalid: unsupported scheme: mailto",
		},
		{
			name: "invalid post link",
			preparePost: func(post *xkcd.Post) {
				post.Link = "https:///"
			},
			errorContains: "post link URL is invalid: URL does not have a host",
		},
		{
			name: "invalid post link scheme",
			preparePost: func(post *xkcd.Post) {
				post.Link = "ssh://lol.com/a"
			},
			errorContains: "post link URL is invalid: unsupported scheme: ssh",
		},
	}

	for _, test := range cases {
		t.Run(
			fmt.Sprintf("invalid post: %s", test.name),
			func(t *testing.T) {
				post, _ := getRandomPost(t, nil, nil)
				test.preparePost(post)
				c := getClient(t, func(_ *http.Request) (*http.Response, error) {
					data, err := json.Marshal(post)
					require.NoError(t, err)
					return &http.Response{
						Status:     "Ok",
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(data)),
					}, nil
				}, nil)
				p, err := c.GetPost(context.Background(), 1)
				assert.Error(t, err, "expected an error")
				assert.Nil(t, p, "expected nil post")
				assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected ErrAPIError")
				assert.ErrorContains(t, err, test.errorContains, "expected error to be valid")
			},
		)
	}
}
