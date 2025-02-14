package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/alitto/pond/v2"

	"github.com/jucrouzet/xkcd/pkg/xkcd"
)

// Update updates the index with posts in range [start..end], with workers concurrent workers.
func (i *Index) Update(ctx context.Context, client *xkcd.Client, start, end, workers uint) error {
	startTime := time.Now()
	logger := i.logger.With(
		slog.Uint64("start", uint64(start)),
		slog.Uint64("end", uint64(end)),
		slog.Uint64("workers", uint64(workers)),
	)
	logger.Debug("updating index")
	tx, errBegin := i.db.BeginTx(ctx, nil)
	if errBegin != nil {
		return fmt.Errorf("failed to start transaction: %w", errBegin)
	}
	if start > end {
		return fmt.Errorf("start must be less than or equal to end")
	}
	if start <= 0 {
		return fmt.Errorf("start must be greater than zero")
	}
	count := new(uint32)
	pool := pond.NewPool(int(workers), pond.WithContext(ctx))
	for num := start; num <= end; num++ {
		pool.SubmitErr(i.handleUpdate(ctx, pool, client, tx, num, count))
	}
	pool.StopAndWait()

	var err error

	defer func(err *error) {
		if *err == nil {
			return
		}
		if errRollback := tx.Rollback(); errRollback != nil && !errors.Is(errRollback, sql.ErrTxDone) {
			logger.Warn("failed to rollback transaction", slog.Any("error", errRollback))
		}
	}(&err)

	dl, ok := ctx.Deadline()
	if ok && dl.Before(time.Now()) {
		err = fmt.Errorf("update failed due to timeout, consider increasing it with `--timeout / -t`")
		return err
	}

	if pool.FailedTasks() > 0 {
		err = errors.New("updating index failed")
		return err
	}

	lastNum := uint(0)
	row := tx.QueryRowContext(ctx, "SELECT max(num) FROM posts")
	if row.Err() != nil && !errors.Is(row.Err(), sql.ErrNoRows) {
		err = fmt.Errorf("failed to get last post number: %w", row.Err())
		return err
	}
	if row.Err() == nil {
		if err := row.Scan(&lastNum); err != nil {
			err = fmt.Errorf("failed to get last post number: %w", err)
			return err
		}
	}

	if _, err = tx.ExecContext(ctx, "UPDATE last_update SET date = ?, last_num = ?", time.Now().Unix(), lastNum); err != nil {
		return fmt.Errorf("failed to update last_update: %w", err)
	}
	if errCommit := tx.Commit(); errCommit != nil {
		return fmt.Errorf("failed to commit transaction: %w", errCommit)
	}
	logger.Debug(
		"finished updating index",
		slog.Duration("duration", time.Since(startTime)),
		slog.Uint64("posts_updated", uint64(atomic.LoadUint32(count))),
	)
	return nil
}

func (i *Index) handleUpdate(ctx context.Context, pool pond.Pool, client *xkcd.Client, tx *sql.Tx, num uint, count *uint32) func() error {
	return func() error {
		if pool.FailedTasks() > 0 {
			return nil
		}
		log := i.logger.With(slog.Uint64("num", uint64(num)))
		log.Debug("updating post")
		post, err := client.GetPost(ctx, num)
		if err != nil {
			log.Warn("failed to get post", slog.String("error", err.Error()))
			return nil
		}
		var data *[]byte
		if i.offline {
			rdr, err := post.GetImageContent(ctx)
			if err != nil {
				log.Warn("failed to get image content", slog.String("error", err.Error()))
				return nil
			}
			defer rdr.Close()
			b, err := io.ReadAll(rdr)
			if err != nil {
				log.Warn("failed to get image content", slog.String("error", err.Error()))
				return nil
			}
			data = &b
		}
		_, err = tx.ExecContext(
			ctx,
			`INSERT OR REPLACE INTO posts
			    (num, title, image, link, date, alt_text, transcript, news, content)
			VALUES
			     (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
			post.Num,
			post.Title,
			post.Img,
			post.Link,
			post.Date.Unix(),
			post.Alt,
			post.Transcript,
			post.News,
			data,
		)
		if err != nil {
			return fmt.Errorf("failed to insert or update post: %w", err)
		}
		atomic.AddUint32(count, 1)
		log.Debug("updated post")
		return nil
	}
}
