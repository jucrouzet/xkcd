package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/BourgeoisBear/rasterm"
	"github.com/fatih/color"
	"github.com/gosuri/uitable"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

// DisplayPost displays a post image.
func DisplayPostImage(
	ctx context.Context,
	out io.Writer,
	post *xkcd.Post,
	d Displayer,
) error {
	img, _, err := post.GetImage(ctx)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	return d(out, img)
}

// DisplayPostInfos displays the infos of a post.
func DisplayPostInfos(out io.Writer, post *xkcd.Post, jsonMode bool) error {
	if jsonMode {
		b, err := json.MarshalIndent(post, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		_, err = out.Write(b)
		return err
	}

	table := uitable.New()
	table.MaxColWidth = 80
	table.Wrap = true // wrap columns

	values := []struct {
		title string
		value string
	}{
		{
			title: "Post number",
			value: fmt.Sprintf("%d", post.Num),
		},
		{
			title: "Title",
			value: post.Title,
		},
		{
			title: "Published on",
			value: post.Date.Format(time.DateOnly),
		},
		{
			title: "URL",
			value: post.Link,
		},
		{
			title: "Image URL",
			value: post.Img,
		},
		{
			title: "Alt text",
			value: post.Alt,
		},
		{
			title: "Transcript",
			value: post.Transcript,
		},
		{
			title: "News",
			value: post.News,
		},
	}
	for _, v := range values {
		if strings.TrimSpace(v.value) == "" {
			continue
		}
		table.AddRow(
			fmt.Sprintf("%s:", v.title),
			color.CyanString(v.value),
		)
	}

	fmt.Fprintln(out, table)
	return nil
}

// Displayer is a function that writes an image to an output writer.
type Displayer func(io.Writer, image.Image) error

// GetDisplayer checks if the terminal supports one of image inline
// protocols and returns the corresponding displayer or nil if it cannot.
func GetDisplayer(logger *slog.Logger) Displayer {
	if rasterm.IsItermCapable() {
		return rasterm.ItermWriteImage
	}
	if rasterm.IsKittyCapable() {
		return func(w io.Writer, img image.Image) error {
			return rasterm.KittyWriteImage(w, img, rasterm.KittyImgOpts{})
		}
	}
	sixel, err := rasterm.IsSixelCapable()
	if err != nil {
		logger.Error("failed to check for sixel support", slog.String("error", err.Error()))
		return nil
	}
	if sixel {
		return func(w io.Writer, img image.Image) error {
			if iPaletted, ok := img.(*image.Paletted); ok {
				return rasterm.SixelWriteImage(w, iPaletted)
			}
			return errors.New("image is not paletted, cannot use sixel to display it")
		}
	}
	return nil
}
