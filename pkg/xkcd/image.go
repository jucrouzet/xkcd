package xkcd

import (
	"context"
	"fmt"
	"image"
	"io"
	"log/slog"
	"net/http"
	"slices"

	// JPEG image format support.
	_ "image/jpeg"
	// PNG image format support.
	_ "image/png"
)

const (
	contentTypeHeader = "Content-Type"
)

// GetImageContent returns a reader to the content of the image associated with the post image.
func (p *Post) GetImageContent(ctx context.Context, client ...HTTPClient) (io.ReadCloser, error) {
	if p.Img == "" {
		return nil, fmt.Errorf("image URL is missing")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.Img, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	p.logger.Debug("fetching image")
	resp, err := p.getClient(client...).Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to send request: %w", ErrAPIError, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected status code: %d", ErrAPIError, resp.StatusCode)
	}
	if !slices.Contains([]string{"image/jpeg", "image/png"}, resp.Header.Get(contentTypeHeader)) {
		return nil, fmt.Errorf("%w: unexpected or undefined content-type: %v", ErrAPIError, resp.Header.Get(contentTypeHeader))
	}
	p.logger.Debug("got image response")
	return resp.Body, nil
}

// GetImage returns an image.Image of the post image.
func (p *Post) GetImage(ctx context.Context, client ...HTTPClient) (image.Image, string, error) {
	data, err := p.GetImageContent(ctx, client...)
	if err != nil {
		return nil, "", err
	}
	defer func(r io.Closer) {
		if err := r.Close(); err != nil {
			p.logger.Warn("failed to close response body", slog.String("error", err.Error()))
		}
	}(data)
	p.logger.Debug("decoding image")
	return image.Decode(data)
}
