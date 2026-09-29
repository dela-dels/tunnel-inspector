package requests

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Repository defines the interface for persisting and querying transactions.
type Repository interface {
	Save(ctx context.Context, tx *HTTPTransaction) error
	GetByID(ctx context.Context, id string) (*HTTPTransaction, error)
	List(ctx context.Context, filter RequestFilter) ([]RequestSummary, int, error)
	DeleteByID(ctx context.Context, id string) error
	Clear(ctx context.Context) error
	Close() error
}

type sqliteRepository struct {
	db *sql.DB
}

// NewSQLiteRepository initializes and returns a SQLite repository.
func NewSQLiteRepository(dbPath string) (Repository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite performance and concurrency
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("failed to set pragma %q: %w", pragma, err)
		}
	}

	// Run migration
	schema := `
	CREATE TABLE IF NOT EXISTS requests (
		id TEXT PRIMARY KEY,
		timestamp DATETIME NOT NULL,
		method TEXT NOT NULL,
		url TEXT NOT NULL,
		path TEXT NOT NULL,
		query TEXT,
		request_headers TEXT,
		request_body BLOB,
		request_size INTEGER,
		response_status INTEGER,
		response_headers TEXT,
		response_body BLOB,
		response_size INTEGER,
		duration_ms INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_requests_timestamp ON requests(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_requests_method ON requests(method);
	CREATE INDEX IF NOT EXISTS idx_requests_path ON requests(path);
	CREATE INDEX IF NOT EXISTS idx_requests_response_status ON requests(response_status);
	`
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return &sqliteRepository{db: db}, nil
}

func (r *sqliteRepository) Save(ctx context.Context, tx *HTTPTransaction) error {
	queryJSON, err := json.Marshal(tx.Request.Query)
	if err != nil {
		queryJSON = []byte("{}")
	}

	reqHeadersJSON, err := json.Marshal(tx.Request.Headers)
	if err != nil {
		reqHeadersJSON = []byte("{}")
	}

	respHeadersJSON, err := json.Marshal(tx.Response.Headers)
	if err != nil {
		respHeadersJSON = []byte("{}")
	}

	query := `
	INSERT INTO requests (
		id, timestamp, method, url, path, query,
		request_headers, request_body, request_size,
		response_status, response_headers, response_body, response_size,
		duration_ms, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	createdAt := tx.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err = r.db.ExecContext(
		ctx,
		query,
		tx.ID,
		tx.Timestamp.UTC(),
		tx.Request.Method,
		tx.Request.URL,
		tx.Request.Path,
		string(queryJSON),
		string(reqHeadersJSON),
		tx.Request.Body,
		tx.Request.Size,
		tx.Response.StatusCode,
		string(respHeadersJSON),
		tx.Response.Body,
		tx.Response.Size,
		tx.Duration,
		createdAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	return nil
}

func (r *sqliteRepository) GetByID(ctx context.Context, id string) (*HTTPTransaction, error) {
	query := `
	SELECT
		id, timestamp, method, url, path, query,
		request_headers, request_body, request_size,
		response_status, response_headers, response_body, response_size,
		duration_ms, created_at
	FROM requests
	WHERE id = ?
	LIMIT 1
	`

	row := r.db.QueryRowContext(ctx, query, id)

	var (
		tx                                          HTTPTransaction
		rawTimestamp, rawCreatedAt                  string
		queryJSON, reqHeadersJSON, respHeadersJSON []byte
	)

	err := row.Scan(
		&tx.ID,
		&rawTimestamp,
		&tx.Request.Method,
		&tx.Request.URL,
		&tx.Request.Path,
		&queryJSON,
		&reqHeadersJSON,
		&tx.Request.Body,
		&tx.Request.Size,
		&tx.Response.StatusCode,
		&respHeadersJSON,
		&tx.Response.Body,
		&tx.Response.Size,
		&tx.Duration,
		&rawCreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query transaction: %w", err)
	}

	tx.Timestamp, _ = parseTimestamp(rawTimestamp)
	tx.CreatedAt, _ = parseTimestamp(rawCreatedAt)

	if len(queryJSON) > 0 {
		_ = json.Unmarshal(queryJSON, &tx.Request.Query)
	}
	if len(reqHeadersJSON) > 0 {
		_ = json.Unmarshal(reqHeadersJSON, &tx.Request.Headers)
	}
	if len(respHeadersJSON) > 0 {
		_ = json.Unmarshal(respHeadersJSON, &tx.Response.Headers)
	}

	return &tx, nil
}

func (r *sqliteRepository) List(ctx context.Context, filter RequestFilter) ([]RequestSummary, int, error) {
	var whereClauses []string
	var args []any

	if filter.Method != "" {
		whereClauses = append(whereClauses, "method = ?")
		args = append(args, strings.ToUpper(filter.Method))
	}
	if filter.Status > 0 {
		whereClauses = append(whereClauses, "response_status = ?")
		args = append(args, filter.Status)
	}
	if filter.Search != "" {
		whereClauses = append(whereClauses, "(path LIKE ? OR url LIKE ?)")
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM requests " + whereSQL
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count requests: %w", err)
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset := (page - 1) * limit

	selectQuery := fmt.Sprintf(`
	SELECT id, timestamp, method, url, path, response_status, duration_ms, request_size, response_size
	FROM requests
	%s
	ORDER BY timestamp DESC
	LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := r.db.QueryContext(ctx, selectQuery, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list requests: %w", err)
	}
	defer rows.Close()

	summaries := make([]RequestSummary, 0)
	for rows.Next() {
		var s RequestSummary
		var rawTimestamp string
		if err := rows.Scan(
			&s.ID,
			&rawTimestamp,
			&s.Method,
			&s.URL,
			&s.Path,
			&s.ResponseStatus,
			&s.Duration,
			&s.RequestSize,
			&s.ResponseSize,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan row: %w", err)
		}
		s.Timestamp, _ = parseTimestamp(rawTimestamp)
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return summaries, total, nil
}

func (r *sqliteRepository) DeleteByID(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM requests WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete request %s: %w", id, err)
	}
	return nil
}

func (r *sqliteRepository) Clear(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM requests")
	if err != nil {
		return fmt.Errorf("failed to clear requests: %w", err)
	}
	return nil
}

func (r *sqliteRepository) Close() error {
	return r.db.Close()
}

func parseTimestamp(val string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, val); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown timestamp format: %s", val)
}
