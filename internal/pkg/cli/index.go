package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	// SQLite driver.
	_ "modernc.org/sqlite"
)

// Index is an index instance.
type Index struct {
	db      *sql.DB
	logger  *slog.Logger
	offline bool
	path    string
}

// NewIndex creates a new index instance.
func NewIndex(path string, logger *slog.Logger) (*Index, error) {
	idx := &Index{
		path:   path,
		logger: logger.With(slog.String("index_path", path)),
	}
	i, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return idx, nil
	}
	if i.IsDir() {
		return nil, errors.New("index file is a directory, not a file")
	}
	idx.logger.Debug("opening index")
	idx.db, err = sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}
	row := idx.db.QueryRow("SELECT value FROM settings WHERE name='offline' LIMIT 1")
	if row.Err() != nil {
		return nil, row.Err()
	}
	var offlineVal string
	if err := row.Scan(&offlineVal); err != nil {
		return nil, row.Err()
	}
	idx.offline = offlineVal == "1"
	idx.logger = idx.logger.With(slog.Bool("offline", idx.offline))
	return idx, err
}

// Initialized returns true if the index is initialized.
func (i *Index) Initized() bool {
	return i.db != nil
}

// Init initializes the index.
func (i *Index) Init(ctx context.Context, force, offline bool) error {
	if i.db != nil && !force {
		return errors.New("index is already initialized")
	}
	if i.db != nil {
		if err := i.db.Close(); err != nil {
			i.logger.With(slog.String("error", err.Error())).Warn("failed to close previous SQLite database")
		}
		if err := os.Remove(i.path); err != nil {
			return fmt.Errorf("failed to remove previous SQLite database: %w", err)
		}
	}
	i.offline = offline
	i.logger.Debug("creating a new SQLite database")
	db, err := sql.Open("sqlite", i.path)
	if err != nil {
		return fmt.Errorf("failed to create a new SQLite database: %w", err)
	}
	_, err = db.ExecContext(
		ctx,
		`create table posts
(
    num        INTEGER NOT NULL
        CONSTRAINT posts_pk
            primary key,
    title      TEXT    NOT NULL,
    image      TEXT    NOT NULL,
    link       TEXT    NOT NULL,
    date       INTEGER NOT NULL,
    alt_text   TEXT    NOT NULL,
    transcript TEXT    NOT NULL,
    news       TEXT    NOT NULL,
	content    BLOB    null,

    CONSTRAINT date_check
        check (date > 0),
    CONSTRAINT num_check
        check (num > 0)
)`,
	)
	if err != nil {
		return fmt.Errorf("failed to create posts table: %w", err)
	}
	_, err = db.ExecContext(ctx, "CREATE INDEX posts_content_index ON posts(title, alt_text, transcript, news);")
	if err != nil {
		return fmt.Errorf("failed to create content index: %w", err)
	}
	_, err = db.ExecContext(ctx, "CREATE TABLE last_update (date INTEGER NOT NULL, last_num INTEGER NOT NULL)")
	if err != nil {
		return fmt.Errorf("failed to create last_update table: %w", err)
	}
	_, err = db.ExecContext(ctx, "CREATE TABLE settings (name TEXT NOT NULL, value TEXT NOT NULL)")
	if err != nil {
		return fmt.Errorf("failed to create settings table: %w", err)
	}
	offlineVal := "0"
	if offline {
		offlineVal = "1"
	}
	_, err = db.ExecContext(ctx, "INSERT INTO settings (name, value) VALUES ('offline', ?)", offlineVal)
	if err != nil {
		return fmt.Errorf("failed to set offline setting: %w", err)
	}
	_, err = db.ExecContext(ctx, "INSERT INTO last_update (date, last_num) VALUES (0, 0)")
	if err != nil {
		return fmt.Errorf("failed to set last_update: %w", err)
	}
	i.logger.Debug("created all tables")
	i.db = db
	return nil
}

// Close closes the index.
func (i *Index) Close() error {
	if i.db != nil {
		return i.db.Close()
	}
	return nil
}

// GetLastUpdate retrieves the last update date and post number from the index.
func (i *Index) GetLastUpdate() (time.Time, uint, error) {
	if i.db == nil {
		return time.Time{}, 0, nil
	}
	row := i.db.QueryRow("SELECT date, last_num FROM last_update LIMIT 1")
	if errors.Is(row.Err(), sql.ErrNoRows) {
		return time.Time{}, 0, nil
	}
	if row.Err() != nil {
		return time.Time{}, 0, row.Err()
	}
	var ts int64
	var num uint
	if err := row.Scan(&ts, &num); err != nil {
		return time.Time{}, 0, row.Err()
	}
	return time.Unix(ts, 0), num, nil
}
