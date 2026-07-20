package search

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"wanwanlog/internal/filter"
	"wanwanlog/internal/logrecord"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	db.SetMaxOpenConns(1)

	store := &SQLiteStore{
		db: db,
	}

	if err := store.Init(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	const createLogsTable = `
CREATE TABLE IF NOT EXISTS logs (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  id TEXT UNIQUE NOT NULL,
  timestamp TEXT,
  service TEXT,
  status TEXT,
  message TEXT,
  attributes TEXT NOT NULL,
  raw TEXT NOT NULL
);
`
	if _, err := s.db.ExecContext(ctx, createLogsTable); err != nil {
		return fmt.Errorf("init sqlite schema: %w", err)
	}

	const createTimestampIndex = `
CREATE INDEX IF NOT EXISTS idx_logs_timestamp ON logs(timestamp);
`

	if _, err := s.db.ExecContext(ctx, createTimestampIndex); err != nil {
		return fmt.Errorf("create timestamp index: %w", err)
	}

	const createServiceIndex = `
CREATE INDEX IF NOT EXISTS idx_logs_service ON logs(service);
`

	if _, err := s.db.ExecContext(ctx, createServiceIndex); err != nil {
		return fmt.Errorf("create service index: %w", err)
	}

	const createStatusIndex = `
CREATE INDEX IF NOT EXISTS idx_logs_status ON logs(status);
`

	if _, err := s.db.ExecContext(ctx, createStatusIndex); err != nil {
		return fmt.Errorf("create status index: %w", err)
	}

	return nil
}

func (s *SQLiteStore) InsertBatch(ctx context.Context, records []logrecord.LogRecord) error {
	if len(records) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	const query = `
INSERT OR IGNORE INTO logs (
  id,
  timestamp,
  service,
  status,
  message,
  attributes,
  raw
) VALUES (?, ?, ?, ?, ?, ?, ?);
`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare insert logs: %w", err)
	}
	defer stmt.Close()

	for _, record := range records {
		attributeJSON, err := json.Marshal(record.Attributes)
		if err != nil {
			return fmt.Errorf("marshal log attributes: %w", err)
		}

		_, err = stmt.ExecContext(
			ctx,
			record.ID,
			formatTimestamp(record.Timestamp),
			record.Service,
			record.Status,
			record.Message,
			string(attributeJSON),
			record.Raw,
		)
		if err != nil {
			return fmt.Errorf("insert log id=%s: %w", record.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit insert logs: %w", err)
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339Nano)
}

func (s *SQLiteStore) Search(ctx context.Context, query filter.SearchQuery) (filter.SearchResult, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}

	offset := max(query.Offset, 0)

	expr, err := filter.Parse(query.Filter)
	if err != nil {
		return filter.SearchResult{}, fmt.Errorf("parse filter: %w", err)
	}

	builder := filter.NewSQLBuilder()

	sqlQuery, err := builder.Build(expr)
	if err != nil {
		return filter.SearchResult{}, fmt.Errorf("build sql: %w", err)
	}

	total, err := s.count(ctx, sqlQuery)
	if err != nil {
		return filter.SearchResult{}, err
	}

	records, err := s.searchRecords(ctx, sqlQuery, limit, offset, query.Order)
	if err != nil {
		return filter.SearchResult{}, err
	}

	return filter.SearchResult{
		Records: records,
		Total:   total,
	}, nil
}

func (s *SQLiteStore) count(ctx context.Context, query filter.SQLQuery) (int, error) {
	sqlText := `
SELECT COUNT(*)
FROM logs
WHERE ` + query.Where

	var total int
	if err := s.db.QueryRowContext(ctx, sqlText, query.Args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count logs: %w", err)
	}

	return total, nil
}

func (s *SQLiteStore) searchRecords(ctx context.Context, query filter.SQLQuery, limit int, offset int, order filter.SearchOrder) ([]logrecord.LogRecord, error) {
	args := append([]any{}, query.Args...)
	args = append(args, limit, offset)

	sqlText := `
SELECT
  id,
  timestamp,
  service,
  status,
  message,
  attributes,
  raw
FROM logs
WHERE ` + query.Where + `
ORDER BY timestamp ASC, sequence ASC
LIMIT ?
OFFSET ?
`
	if order == filter.LatestWindow {
		sqlText = `
SELECT id, timestamp, service, status, message, attributes, raw
FROM (
  SELECT sequence, id, timestamp, service, status, message, attributes, raw
  FROM logs
  WHERE ` + query.Where + `
  ORDER BY timestamp DESC, sequence DESC
  LIMIT ? OFFSET ?
)
ORDER BY timestamp ASC, sequence ASC`
	}

	rows, err := s.db.QueryContext(ctx, sqlText, args...)
	if err != nil {
		return nil, fmt.Errorf("search logs: %w", err)
	}
	defer rows.Close()

	var records []logrecord.LogRecord

	for rows.Next() {
		record, err := scanLogRecord(rows)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate logs: %w", err)
	}

	return records, nil
}

func (s *SQLiteStore) DeleteAll(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "DELETE FROM logs"); err != nil {
		return fmt.Errorf("delete all logs: %w", err)
	}
	return nil
}

func scanLogRecord(rows *sql.Rows) (logrecord.LogRecord, error) {
	var (
		record        logrecord.LogRecord
		timestampText string
		attributesRaw string
	)

	if err := rows.Scan(
		&record.ID,
		&timestampText,
		&record.Service,
		&record.Status,
		&record.Message,
		&attributesRaw,
		&record.Raw,
	); err != nil {
		return logrecord.LogRecord{}, fmt.Errorf("scan log record: %w", err)
	}

	if timestampText != "" {
		timestamp, err := time.Parse(time.RFC3339Nano, timestampText)
		if err != nil {
			return logrecord.LogRecord{}, fmt.Errorf("parse timestamp %q: %w", timestampText, err)
		}

		record.Timestamp = timestamp
	}

	var attributes map[string]any
	if err := json.Unmarshal([]byte(attributesRaw), &attributes); err != nil {
		return logrecord.LogRecord{}, fmt.Errorf("unmarshal attributes: %w", err)
	}

	record.Attributes = attributes

	return record, nil
}
