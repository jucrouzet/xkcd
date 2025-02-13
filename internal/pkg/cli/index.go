package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	// SQLite driver.
	_ "github.com/glebarez/go-sqlite"
)

// Index is an index instance.
type Index struct {
	db     *sql.DB
	logger *slog.Logger
	path   string
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
	return idx, err
}

// Initialized returns true if the index is initialized.
func (i *Index) Initized() bool {
	return i.db != nil
}

// Init initializes the index.
func (i *Index) Init(force bool) error {
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
	i.logger.Debug("creating a new SQLite database")
	db, err := sql.Open("sqlite", i.path)
	if err != nil {
		return fmt.Errorf("failed to create a new SQLite database: %w", err)
	}
	_, err = db.Exec(`
create table posts
(
    num        INTEGER not null
        constraint posts_pk
            primary key,
    title      TEXT    not null,
    image      TEXT    not null,
    link       TEXT    not null,
    date       integer not null,
    alt_text   TEXT    not null,
    transcript TEXT    not null,
    news       TEXT    not null,
    constraint date_check
        check (date > 0),
    constraint num_check
        check (num > 0)
)`)
	if err != nil {
		return fmt.Errorf("failed to create posts table: %w", err)
	}
	_, err = db.Exec("create index posts_content_index on posts (title, alt_text, transcript, news);")
	if err != nil {
		return fmt.Errorf("failed to create content index: %w", err)
	}
	_, err = db.Exec(`create table last_check (date integer not null)`)
	if err != nil {
		return fmt.Errorf("failed to create last_check table: %w", err)
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

// GetLastUpdate retrieves the last update date from the index.
func (i *Index) GetLastUpdate() (time.Time, error) {
	if i.db == nil {
		return time.Time{}, nil
	}
	row := i.db.QueryRow("SELECT date FROM last_check LIMIT 1")
	if errors.Is(row.Err(), sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if row.Err() != nil {
		return time.Time{}, row.Err()
	}
	var ts int64
	if err := row.Scan(&ts); err != nil {
		return time.Time{}, row.Err()
	}
	return time.Unix(ts, 0), nil
}
