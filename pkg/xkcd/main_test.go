package xkcd_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-faker/faker/v4"
	slogmock "github.com/samber/slog-mock"
	"github.com/stretchr/testify/require"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

type mockClient func(*http.Request) (*http.Response, error)

func getClient(
	t testing.TB,
	mockCall mockClient,
	loggerHandler func(ctx context.Context, record slog.Record) error,
) *xkcd.Client {
	t.Helper()

	if loggerHandler == nil {
		loggerHandler = func(_ context.Context, _ slog.Record) error {
			return nil
		}
	}

	return xkcd.New(
		xkcd.WithClient(&http.Client{Transport: &mockRoundTripper{
			mock: mockCall,
			t:    t,
		}}),
		xkcd.WithLogger(slog.New(
			slogmock.Option{
				Handle: loggerHandler,
			}.NewMockHandler(),
		)),
	)
}

func sendValidPost(t testing.TB) (*http.Response, error) {
	t.Helper()
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(`{"month": "1", "num": 1, "link": "", "year": "2006", "news": "", "safe_title": "Barrel - Part 1", "transcript": "[[A boy sits in a barrel which is floating in an ocean.]]\nBoy: I wonder where I'll float next?\n[[The barrel drifts into the distance. Nothing else can be seen.]]\n{{Alt: Don't we all.}}", "alt": "Don't we all.", "img": "https://imgs.xkcd.com/comics/barrel_cropped_(1).jpg", "title": "Barrel - Part 1", "day": "1"}`)),
	}, nil
}

type mockRoundTripper struct {
	mock mockClient
	t    testing.TB
}

func (rt *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.mock == nil {
		return sendValidPost(rt.t)
	}
	return rt.mock(req)
}

func getRandomPost(t testing.TB, errorRead error, errorClose error) (*xkcd.Post, *http.Response) {
	date, err := time.ParseInLocation(time.DateOnly, faker.Date(), time.Local)
	require.NoError(t, err)
	post := &xkcd.Post{
		Alt:        randomEmpty(t, faker.Sentence()),
		Date:       date,
		Day:        fmt.Sprintf("%d", date.Day()),
		Img:        faker.URL(),
		Link:       randomEmpty(t, faker.URL()),
		Month:      fmt.Sprintf("%d", date.Month()),
		News:       randomEmpty(t, faker.Paragraph()),
		Num:        rand.Uint() + 1,
		SafeTitle:  randomEmpty(t, faker.Sentence()),
		Title:      faker.Sentence(),
		Transcript: randomEmpty(t, faker.Paragraph()),
		Year:       fmt.Sprintf("%d", date.Year()),
	}
	dat, err := json.Marshal(post)
	require.NoError(t, err)
	resp := &http.Response{
		Status:     "OK",
		StatusCode: http.StatusOK,
		Body: &fakeReader{
			data:       io.NopCloser(strings.NewReader(string(dat))),
			errorClose: errorClose,
			errorRead:  errorRead,
			t:          t,
		},
	}
	return post, resp
}

func randomEmpty(t testing.TB, v string) string {
	t.Helper()
	if rand.IntN(100) < 75 {
		return v
	}
	return ""
}

func getImageResponse(t testing.TB, imgType string, errorRead error, errorClose error) *http.Response {
	t.Helper()
	var file string
	var header string
	resp := &http.Response{
		Body:       io.NopCloser(http.NoBody),
		Status:     "OK",
		StatusCode: http.StatusOK,
	}
	switch imgType {
	case "jpg":
		file = "testdata/image.jpg"
		header = "image/jpeg"
	case "png":
		file = "testdata/image.png"
		header = "image/png"
	case "audio":
		header = "audio/mpeg"
	case "bogus":
		resp.Body = io.NopCloser(strings.NewReader("bogus data"))
		header = "image/jpeg"
	case "error":
		header = "image/jpeg"
		resp.Status = "Internal Server Error"
		resp.StatusCode = http.StatusInternalServerError
	default:
		t.Fatalf("Unsupported image type: %s", imgType)
	}
	if file != "" {
		f, err := os.Open(file)
		require.NoError(t, err)
		rdr := &fakeReader{
			data:       f,
			errorClose: errorClose,
			errorRead:  errorRead,
			t:          t,
		}
		resp.Body = rdr
	}
	hdr := http.Header{}
	hdr.Set("content-type", header)
	resp.Header = hdr
	return resp
}

type fakeReader struct {
	data       io.ReadCloser
	errorRead  error
	errorClose error
	t          testing.TB
}

func (fr *fakeReader) Read(p []byte) (n int, err error) {
	fr.t.Helper()
	if fr.errorRead != nil {
		return 0, fr.errorRead
	}
	return fr.data.Read(p)
}

func (fr *fakeReader) Close() error {
	fr.t.Helper()
	if fr.errorClose != nil {
		return fr.errorClose
	}
	return fr.data.Close()
}
