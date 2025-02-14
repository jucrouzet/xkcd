package xkcd_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

func TestPost_GetImageContent(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path jpeg", func(t *testing.T) {
		imgResp := getImageResponse(t, "jpg", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, rdr, "expected non-nil reader")
		data, err := io.ReadAll(rdr)
		assert.NoError(t, err, "expected read to not return an error")
		assert.Len(t, data, 24848, "expected data to be read correctly")
	})

	t.Run("happy path png", func(t *testing.T) {
		imgResp := getImageResponse(t, "png", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, rdr, "expected non-nil reader")
		data, err := io.ReadAll(rdr)
		assert.NoError(t, err, "expected read to not return an error")
		assert.Len(t, data, 54845, "expected data to be read correctly")
	})

	t.Run("happy path gif", func(t *testing.T) {
		imgResp := getImageResponse(t, "gif", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, rdr, "expected non-nil reader")
		data, err := io.ReadAll(rdr)
		assert.NoError(t, err, "expected read to not return an error")
		assert.Len(t, data, 56039, "expected data to be read correctly")
	})

	t.Run("image url is empty", func(t *testing.T) {
		_, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(_ *http.Request) (*http.Response, error) {
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		p.Img = ""
		rdr, err := p.GetImageContent(context.Background())
		require.Nil(t, rdr, "expected nil reader")
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorContains(t, err, "image URL is missing", "expected error to have correct message")
	})

	t.Run("failed to make request", func(t *testing.T) {
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return nil, errors.New("kaboom ?")
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.Nil(t, rdr, "expected nil reader")
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected error to be ErrAPIError")
		assert.ErrorContains(t, err, "failed to send request", "expected error to have correct message")
	})

	t.Run("invalid status code", func(t *testing.T) {
		imgResp := getImageResponse(t, "error", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.Nil(t, rdr, "expected nil reader")
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected error to be ErrAPIError")
		assert.ErrorContains(t, err, "unexpected status code: 500", "expected error to have correct message")
	})

	t.Run("invalid content type", func(t *testing.T) {
		imgResp := getImageResponse(t, "audio", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.Nil(t, rdr, "expected nil reader")
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected error to be ErrAPIError")
		assert.ErrorContains(t, err, "unexpected or undefined content-type: audio/mpeg", "expected error to have correct message")
	})

	t.Run("error while reading", func(t *testing.T) {
		imgResp := getImageResponse(t, "png", errors.New("yes, rico, kaboom"), nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		rdr, err := p.GetImageContent(context.Background())
		require.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, rdr, "expected non-nil reader")
		_, err = io.ReadAll(rdr)
		assert.Error(t, err, "expected read to return an error")
		assert.ErrorContains(t, err, "yes, rico, kaboom", "expected original error to be returned")
	})
}

func TestPost_GetImage(t *testing.T) {
	ctx := context.Background()

	t.Run("happy path jpeg", func(t *testing.T) {
		closeCalled := false
		imgResp := getImageResponse(t, "jpg", nil, errors.New("cannot close"))
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(
			t,
			func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == expectedPost.Img {
					return imgResp, nil
				}
				return resp, nil
			},
			func(_ context.Context, record slog.Record) error {
				if record.Level == slog.LevelWarn && record.Message == "failed to close response body" {
					closeCalled = true
				}
				return nil
			},
		)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		img, imgType, err := p.GetImage(context.Background())
		assert.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, img, "expected non-nil image")
		assert.Equal(t, "jpeg", imgType, "expected image to be decoded correctly")
		assert.Equal(t, 577, img.Bounds().Dx(), "expected image to be decoded correctly")
		assert.True(t, closeCalled, "expected close to have been called on image response body")
	})

	t.Run("happy path png", func(t *testing.T) {
		closeCalled := false
		imgResp := getImageResponse(t, "png", nil, errors.New("cannot close"))
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(
			t,
			func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == expectedPost.Img {
					return imgResp, nil
				}
				return resp, nil
			},
			func(_ context.Context, record slog.Record) error {
				if record.Level == slog.LevelWarn && record.Message == "failed to close response body" {
					closeCalled = true
				}
				return nil
			},
		)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		img, imgType, err := p.GetImage(context.Background())
		assert.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, img, "expected non-nil image")
		assert.Equal(t, "png", imgType, "expected image to be decoded correctly")
		assert.Equal(t, 281, img.Bounds().Dx(), "expected image to be decoded correctly")
		assert.True(t, closeCalled, "expected close to have been called on image response body")
	})

	t.Run("happy path gif", func(t *testing.T) {
		closeCalled := false
		imgResp := getImageResponse(t, "gif", nil, errors.New("cannot close"))
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(
			t,
			func(r *http.Request) (*http.Response, error) {
				if r.URL.String() == expectedPost.Img {
					return imgResp, nil
				}
				return resp, nil
			},
			func(_ context.Context, record slog.Record) error {
				if record.Level == slog.LevelWarn && record.Message == "failed to close response body" {
					closeCalled = true
				}
				return nil
			},
		)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		img, imgType, err := p.GetImage(context.Background())
		assert.NoError(t, err, "expected no error while getting image")
		require.NotNil(t, img, "expected non-nil image")
		assert.Equal(t, "gif", imgType, "expected image to be decoded correctly")
		assert.Equal(t, 713, img.Bounds().Dx(), "expected image to be decoded correctly")
		assert.True(t, closeCalled, "expected close to have been called on image response body")
	})

	t.Run("error while getting image content", func(t *testing.T) {
		imgResp := getImageResponse(t, "error", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		img, imgType, err := p.GetImage(context.Background())
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorIs(t, err, xkcd.ErrAPIError, "expected an ErrAPIError error")
		assert.ErrorContains(t, err, "unexpected status code: 500", "expected original error to be wrapped")
		assert.Nil(t, img, "expected nil image")
		assert.Empty(t, imgType, "expected empty image type")
	})

	t.Run("bogus image", func(t *testing.T) {
		imgResp := getImageResponse(t, "bogus", nil, nil)
		defer imgResp.Body.Close()
		expectedPost, resp := getRandomPost(t, nil, nil)
		c := getClient(t, func(r *http.Request) (*http.Response, error) {
			if r.URL.String() == expectedPost.Img {
				return imgResp, nil
			}
			return resp, nil
		}, nil)
		p, err := c.GetPost(ctx, 1)
		require.NoError(t, err, "expected no error while getting post")
		require.NotNil(t, p, "expected non-nil post")
		img, imgType, err := p.GetImage(context.Background())
		assert.Error(t, err, "expected error while getting image")
		assert.ErrorContains(t, err, "image: unknown format", "expected original error to be wrapped")
		assert.Nil(t, img, "expected nil image")
		assert.Empty(t, imgType, "expected empty image type")
	})
}
