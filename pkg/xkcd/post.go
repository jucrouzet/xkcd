package xkcd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Post represents a comic post from xkcd.com.
type Post struct {
	// Alt is the alternative text for the post image.
	Alt string `json:"alt"`
	// Date is the publication date of the post.
	Date time.Time
	// Day is the day of the month of the publication date of the post as string.
	Day string `json:"day"`
	// Img is the URL of the post image.
	Img string `json:"img"`
	// Link is the URL of the post page.
	Link string `json:"link"`
	// Month is the month number of the publication date of the post as string.
	Month string `json:"month"`
	// News is the news text published with the post.
	News string `json:"news"`
	// Num is the number of the post.
	Num uint `json:"num"`
	// SafeTitle is the safe title for the post.
	SafeTitle string `json:"safe_title"`
	// Title is the title of the post.
	Title string `json:"title"`
	// Transcript is the transcript for the post.
	Transcript string `json:"transcript"`
	// Year is the year of the publication date of the post as string.
	Year string `json:"year"`

	defaultClient HTTPClient
	logger        *slog.Logger
}

var (
	// ErrNoSuchPost is returned when a requested post does not exist.
	ErrNoSuchPost = errors.New("no such post")
	// ErrAPIError is returned when xkcd api returned an error.
	ErrAPIError = errors.New("xkcd API error")
)

// NewPost returns a new xkcd API client with the provided options.
func NewPost(defaultClient HTTPClient, logger *slog.Logger) *Post {
	return &Post{
		defaultClient: defaultClient,
		logger:        logger,
	}
}

// GetLatest retrieves the latest post.
func (c *Client) GetLatest(ctx context.Context, client ...HTTPClient) (*Post, error) {
	return c.getPost(ctx, "https://xkcd.com/info.0.json", client...)
}

// GetPost retrieves the post with the given number.
func (c *Client) GetPost(ctx context.Context, num uint, client ...HTTPClient) (*Post, error) {
	if num == 0 {
		return nil, ErrNoSuchPost
	}
	return c.getPost(ctx, fmt.Sprintf("https://xkcd.com/%d/info.0.json", num), client...)
}

func (c *Client) getPost(ctx context.Context, apiURL string, client ...HTTPClient) (*Post, error) {
	logger := c.logger.With(slog.String("url", apiURL))
	logger.Debug("fetching post")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	//nolint: bodyclose // Body is closed in the defer below
	resp, err := c.getClient(client...).Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to send request: %w", ErrAPIError, err)
	}
	defer func(body io.ReadCloser) {
		_, _ = io.Copy(io.Discard, body)
		if err := body.Close(); err != nil {
			c.logger.Warn("failed to close response body", slog.String("error", err.Error()))
		}
	}(resp.Body)
	logger.Debug("got api response")
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoSuchPost
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status code is %d", ErrAPIError, resp.StatusCode)
	}

	var post *Post
	err = json.NewDecoder(resp.Body).Decode(&post)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to decode response: %w", ErrAPIError, err)
	}
	post.defaultClient = c.defaultClient
	post.logger = logger
	return parsePost(post)
}

func parsePost(post *Post) (*Post, error) {
	if post.Num == 0 {
		return nil, fmt.Errorf("%w: post number is zero", ErrAPIError)
	}
	day, err := strconv.Atoi(post.Day)
	if err == nil && (day < 1 || day > 31) {
		err = fmt.Errorf("invalid value for day: %d", day)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse date day: %w", ErrAPIError, err)
	}
	month, err := strconv.Atoi(post.Month)
	if err == nil && (month < int(time.January) || month > int(time.December)) {
		err = fmt.Errorf("invalid value for month: %d", month)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse date month: %w", ErrAPIError, err)
	}
	year, err := strconv.Atoi(post.Year)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to parse date year: %w", ErrAPIError, err)
	}

	if err = validateURL(post.Img); err != nil {
		return nil, fmt.Errorf("%w: post image URL is invalid: %w", ErrAPIError, err)
	}

	if post.Link != "" {
		if err = validateURL(post.Link); err != nil {
			return nil, fmt.Errorf("%w: post link URL is invalid: %w", ErrAPIError, err)
		}
	} else {
		post.Link = fmt.Sprintf("https://xkcd.com/%d/", post.Num)
	}

	post.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
	return post, nil
}

func validateURL(u string) error {
	if strings.TrimSpace(u) == "" {
		return errors.New("URL is empty")
	}
	v, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid syntax")
	}
	if v.Scheme != "http" && v.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", v.Scheme)
	}
	if v.Host == "" {
		return errors.New("URL does not have a host")
	}
	return nil
}
