package cli

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

// Get gets a post.
// If it exists in index, it returns it.
// Else, it fetches it from xkcd API client and stores it in index.
func (i *Index) Get(ctx context.Context, client *xkcd.Client, num uint) (*xkcd.Post, error) {
	post, err := i.getFromIndex(ctx, num)
	if err != nil {
		return nil, err
	}
	if post != nil {
		return post, nil
	}
	post, err = client.GetPost(ctx, num)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch post from API: %w", err)
	}
	if err = i.indexPost(ctx, post, i.db, i.logger); err != nil {
		i.logger.Warn("failed to index post", slog.Any("error", err))
	}
	return post, nil
}

func (i *Index) getFromIndex(ctx context.Context, num uint) (*xkcd.Post, error) {
	if i.db == nil {
		return nil, nil
	}
	row := i.db.QueryRowContext(ctx, "SELECT * FROM posts WHERE num = ?", num)
	if row.Err() != nil {
		return nil, fmt.Errorf("failed to search post in index: %w", row.Err())
	}

	getter := &getter{
		logger: i.logger,
	}
	post := xkcd.NewPost(
		getter,
		i.logger,
	)

	var ts int64
	var data *[]byte

	if err := row.Scan(
		&post.Num,
		&post.Title,
		&post.Img,
		&post.Link,
		&ts,
		&post.Alt,
		&post.Transcript,
		&post.News,
		&data,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			i.logger.Debug("post not found in index")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}
	i.logger.Debug("post found in index")
	post.Date = time.Unix(ts, 0)
	if data != nil {
		getter.data = *data
	}
	return post, nil
}

type getter struct {
	data   []byte
	logger *slog.Logger
}

// Do implements the xkcd.HTTPClient interface.
func (g *getter) Do(r *http.Request) (*http.Response, error) {
	if g.data == nil {
		g.logger.Debug("image was indexed online, serving from HTTP")
		return http.DefaultClient.Do(r)
	}
	g.logger.Debug("image was indexed offline, serving from index")
	headers := http.Header{}
	headers.Set("Content-Type", "image/jpeg")
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(g.data)),
		Header:     headers,
	}, nil
}
