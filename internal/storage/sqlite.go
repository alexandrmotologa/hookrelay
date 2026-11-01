package storage

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var initSQL string

type SQLiteStorage struct {
	db *sql.DB
}

// NewSQLiteStorage initializes SQLite database with WAL configuration.
func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	dsn := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", dbPath)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	s := &SQLiteStorage{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return s, nil
}

func (s *SQLiteStorage) migrate() error {
	if _, err := s.db.Exec(initSQL); err != nil {
		return err
	}

	// Idempotent column additions for existing databases
	_, _ = s.db.Exec(`ALTER TABLE endpoints ADD COLUMN previous_secret TEXT;`)
	_, _ = s.db.Exec(`ALTER TABLE endpoints ADD COLUMN secret_rotated_at DATETIME;`)
	_, _ = s.db.Exec(`ALTER TABLE endpoints ADD COLUMN custom_headers TEXT NOT NULL DEFAULT '{}';`)

	return nil
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// =========================================================================
// Endpoints
// =========================================================================

func (s *SQLiteStorage) CreateEndpoint(ctx context.Context, ep *Endpoint) error {
	if ep.ID == "" {
		ep.ID = "ep_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	}
	now := time.Now().UTC()
	ep.CreatedAt = now
	ep.UpdatedAt = now
	if ep.Status == "" {
		ep.Status = EndpointStatusActive
	}
	if ep.RateLimit == 0 {
		ep.RateLimit = 100
	}

	eventTypesJSON := EncodeEventTypes(ep.EventTypes)
	customHeadersJSON := EncodeCustomHeaders(ep.CustomHeaders)

	query := `
		INSERT INTO endpoints (
			id, name, url, secret, previous_secret, secret_rotated_at,
			event_types, custom_headers, rate_limit, status,
			consecutive_failures, tripped_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		ep.ID, ep.Name, ep.URL, ep.Secret, ep.PreviousSecret, ep.SecretRotatedAt,
		eventTypesJSON, customHeadersJSON, ep.RateLimit, string(ep.Status),
		ep.ConsecutiveFailures, ep.TrippedAt, ep.CreatedAt, ep.UpdatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetEndpoint(ctx context.Context, id string) (*Endpoint, error) {
	query := `
		SELECT id, name, url, secret, COALESCE(previous_secret, ''), secret_rotated_at,
		       event_types, COALESCE(custom_headers, '{}'), rate_limit, status,
		       consecutive_failures, tripped_at, created_at, updated_at
		FROM endpoints WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var ep Endpoint
	var eventTypesJSON string
	var customHeadersJSON string
	var statusStr string
	err := row.Scan(
		&ep.ID, &ep.Name, &ep.URL, &ep.Secret, &ep.PreviousSecret, &ep.SecretRotatedAt,
		&eventTypesJSON, &customHeadersJSON, &ep.RateLimit, &statusStr,
		&ep.ConsecutiveFailures, &ep.TrippedAt, &ep.CreatedAt, &ep.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	ep.Status = EndpointStatus(statusStr)
	ep.EventTypes = DecodeEventTypes(eventTypesJSON)
	ep.CustomHeaders = DecodeCustomHeaders(customHeadersJSON)
	return &ep, nil
}

func (s *SQLiteStorage) ListEndpoints(ctx context.Context) ([]*Endpoint, error) {
	query := `
		SELECT id, name, url, secret, COALESCE(previous_secret, ''), secret_rotated_at,
		       event_types, COALESCE(custom_headers, '{}'), rate_limit, status,
		       consecutive_failures, tripped_at, created_at, updated_at
		FROM endpoints ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*Endpoint
	for rows.Next() {
		var ep Endpoint
		var eventTypesJSON string
		var customHeadersJSON string
		var statusStr string
		err := rows.Scan(
			&ep.ID, &ep.Name, &ep.URL, &ep.Secret, &ep.PreviousSecret, &ep.SecretRotatedAt,
			&eventTypesJSON, &customHeadersJSON, &ep.RateLimit, &statusStr,
			&ep.ConsecutiveFailures, &ep.TrippedAt, &ep.CreatedAt, &ep.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		ep.Status = EndpointStatus(statusStr)
		ep.EventTypes = DecodeEventTypes(eventTypesJSON)
		ep.CustomHeaders = DecodeCustomHeaders(customHeadersJSON)
		result = append(result, &ep)
	}
	return result, rows.Err()
}

func (s *SQLiteStorage) UpdateEndpoint(ctx context.Context, ep *Endpoint) error {
	ep.UpdatedAt = time.Now().UTC()
	eventTypesJSON := EncodeEventTypes(ep.EventTypes)
	customHeadersJSON := EncodeCustomHeaders(ep.CustomHeaders)

	query := `
		UPDATE endpoints
		SET name = ?, url = ?, secret = ?, previous_secret = ?, secret_rotated_at = ?,
		    event_types = ?, custom_headers = ?, rate_limit = ?, status = ?,
		    consecutive_failures = ?, tripped_at = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		ep.Name, ep.URL, ep.Secret, ep.PreviousSecret, ep.SecretRotatedAt,
		eventTypesJSON, customHeadersJSON, ep.RateLimit, string(ep.Status),
		ep.ConsecutiveFailures, ep.TrippedAt, ep.UpdatedAt, ep.ID,
	)
	return err
}

func (s *SQLiteStorage) DeleteEndpoint(ctx context.Context, id string) error {
	query := `DELETE FROM endpoints WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

func (s *SQLiteStorage) IncrementEndpointFailure(ctx context.Context, id string, threshold int) (bool, error) {
	ep, err := s.GetEndpoint(ctx, id)
	if err != nil {
		return false, err
	}

	ep.ConsecutiveFailures++
	tripped := false
	if ep.ConsecutiveFailures >= threshold && ep.Status == EndpointStatusActive {
		ep.Status = EndpointStatusSuspended
		now := time.Now().UTC()
		ep.TrippedAt = &now
		tripped = true
	}

	return tripped, s.UpdateEndpoint(ctx, ep)
}

func (s *SQLiteStorage) ResetEndpointFailure(ctx context.Context, id string) error {
	ep, err := s.GetEndpoint(ctx, id)
	if err != nil {
		return err
	}
	ep.ConsecutiveFailures = 0
	ep.TrippedAt = nil
	if ep.Status == EndpointStatusSuspended {
		ep.Status = EndpointStatusActive
	}
	return s.UpdateEndpoint(ctx, ep)
}

func (s *SQLiteStorage) GetActiveEndpointsForEvent(ctx context.Context, eventType string) ([]*Endpoint, error) {
	all, err := s.ListEndpoints(ctx)
	if err != nil {
		return nil, err
	}
	var matched []*Endpoint
	for _, ep := range all {
		if ep.Status == EndpointStatusActive && ep.MatchesEvent(eventType) {
			matched = append(matched, ep)
		}
	}
	return matched, nil
}

// =========================================================================
// Messages
// =========================================================================

func (s *SQLiteStorage) CreateMessage(ctx context.Context, msg *Message) error {
	if msg.ID == "" {
		msg.ID = "msg_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO messages (id, idempotency_key, event_type, payload, created_at)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query, msg.ID, msg.IdempotencyKey, msg.EventType, msg.Payload, msg.CreatedAt)
	return err
}

func (s *SQLiteStorage) GetMessage(ctx context.Context, id string) (*Message, error) {
	query := `SELECT id, idempotency_key, event_type, payload, created_at FROM messages WHERE id = ?`
	row := s.db.QueryRowContext(ctx, query, id)

	var msg Message
	err := row.Scan(&msg.ID, &msg.IdempotencyKey, &msg.EventType, &msg.Payload, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (s *SQLiteStorage) GetMessageByIdempotencyKey(ctx context.Context, key string) (*Message, error) {
	query := `SELECT id, idempotency_key, event_type, payload, created_at FROM messages WHERE idempotency_key = ?`
	row := s.db.QueryRowContext(ctx, query, key)

	var msg Message
	err := row.Scan(&msg.ID, &msg.IdempotencyKey, &msg.EventType, &msg.Payload, &msg.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

// =========================================================================
// Deliveries
// =========================================================================

func (s *SQLiteStorage) CreateDelivery(ctx context.Context, d *Delivery) error {
	if d.ID == "" {
		d.ID = "del_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16]
	}
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	if d.Status == "" {
		d.Status = DeliveryStatusPending
	}
	if d.MaxAttempts == 0 {
		d.MaxAttempts = 5
	}

	query := `
		INSERT INTO deliveries (
			id, message_id, endpoint_id, status, attempt_number, max_attempts,
			http_status, duration_ms, request_headers, response_headers,
			response_body, error_message, next_retry_at, sent_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		d.ID, d.MessageID, d.EndpointID, string(d.Status), d.AttemptNumber, d.MaxAttempts,
		d.HTTPStatus, d.DurationMs, d.RequestHeaders, d.ResponseHeaders,
		d.ResponseBody, d.ErrorMessage, d.NextRetryAt, d.SentAt, d.CreatedAt,
	)
	return err
}

func (s *SQLiteStorage) GetDelivery(ctx context.Context, id string) (*Delivery, error) {
	query := `
		SELECT id, message_id, endpoint_id, status, attempt_number, max_attempts,
		       http_status, duration_ms, request_headers, response_headers,
		       response_body, error_message, next_retry_at, sent_at, created_at
		FROM deliveries WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var d Delivery
	var statusStr string
	err := row.Scan(
		&d.ID, &d.MessageID, &d.EndpointID, &statusStr, &d.AttemptNumber, &d.MaxAttempts,
		&d.HTTPStatus, &d.DurationMs, &d.RequestHeaders, &d.ResponseHeaders,
		&d.ResponseBody, &d.ErrorMessage, &d.NextRetryAt, &d.SentAt, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.Status = DeliveryStatus(statusStr)
	return &d, nil
}

func (s *SQLiteStorage) GetDeliveryWithDetails(ctx context.Context, id string) (*DeliveryWithDetails, error) {
	query := `
		SELECT d.id, d.message_id, d.endpoint_id, d.status, d.attempt_number, d.max_attempts,
		       d.http_status, d.duration_ms, d.request_headers, d.response_headers,
		       d.response_body, d.error_message, d.next_retry_at, d.sent_at, d.created_at,
		       e.name, e.url, m.event_type, m.payload
		FROM deliveries d
		JOIN endpoints e ON d.endpoint_id = e.id
		JOIN messages m ON d.message_id = m.id
		WHERE d.id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var d DeliveryWithDetails
	var statusStr string
	err := row.Scan(
		&d.ID, &d.MessageID, &d.EndpointID, &statusStr, &d.AttemptNumber, &d.MaxAttempts,
		&d.HTTPStatus, &d.DurationMs, &d.RequestHeaders, &d.ResponseHeaders,
		&d.ResponseBody, &d.ErrorMessage, &d.NextRetryAt, &d.SentAt, &d.CreatedAt,
		&d.EndpointName, &d.EndpointURL, &d.EventType, &d.Payload,
	)
	if err != nil {
		return nil, err
	}
	d.Status = DeliveryStatus(statusStr)
	return &d, nil
}

func (s *SQLiteStorage) UpdateDeliveryAttempt(ctx context.Context, d *Delivery) error {
	query := `
		UPDATE deliveries
		SET status = ?, attempt_number = ?, http_status = ?, duration_ms = ?,
		    request_headers = ?, response_headers = ?, response_body = ?,
		    error_message = ?, next_retry_at = ?, sent_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		string(d.Status), d.AttemptNumber, d.HTTPStatus, d.DurationMs,
		d.RequestHeaders, d.ResponseHeaders, d.ResponseBody,
		d.ErrorMessage, d.NextRetryAt, d.SentAt, d.ID,
	)
	return err
}

func (s *SQLiteStorage) ListDeliveries(ctx context.Context, filter DeliveryFilter) ([]*DeliveryWithDetails, int64, error) {
	var whereClauses []string
	var args []any

	if filter.EndpointID != "" {
		whereClauses = append(whereClauses, "d.endpoint_id = ?")
		args = append(args, filter.EndpointID)
	}
	if filter.Status != "" {
		whereClauses = append(whereClauses, "d.status = ?")
		args = append(args, filter.Status)
	}
	if filter.EventType != "" {
		whereClauses = append(whereClauses, "m.event_type = ?")
		args = append(args, filter.EventType)
	}
	if filter.Search != "" {
		whereClauses = append(whereClauses, "(d.id LIKE ? OR m.id LIKE ? OR m.payload LIKE ? OR e.url LIKE ?)")
		pattern := "%" + filter.Search + "%"
		args = append(args, pattern, pattern, pattern, pattern)
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM deliveries d
		JOIN endpoints e ON d.endpoint_id = e.id
		JOIN messages m ON d.message_id = m.id
		%s
	`, whereSQL)

	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT d.id, d.message_id, d.endpoint_id, d.status, d.attempt_number, d.max_attempts,
		       d.http_status, d.duration_ms, d.request_headers, d.response_headers,
		       d.response_body, d.error_message, d.next_retry_at, d.sent_at, d.created_at,
		       e.name, e.url, m.event_type, m.payload
		FROM deliveries d
		JOIN endpoints e ON d.endpoint_id = e.id
		JOIN messages m ON d.message_id = m.id
		%s
		ORDER BY d.created_at DESC
		LIMIT ? OFFSET ?
	`, whereSQL)

	queryArgs := append(args, limit, offset)
	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var list []*DeliveryWithDetails
	for rows.Next() {
		var d DeliveryWithDetails
		var statusStr string
		err := rows.Scan(
			&d.ID, &d.MessageID, &d.EndpointID, &statusStr, &d.AttemptNumber, &d.MaxAttempts,
			&d.HTTPStatus, &d.DurationMs, &d.RequestHeaders, &d.ResponseHeaders,
			&d.ResponseBody, &d.ErrorMessage, &d.NextRetryAt, &d.SentAt, &d.CreatedAt,
			&d.EndpointName, &d.EndpointURL, &d.EventType, &d.Payload,
		)
		if err != nil {
			return nil, 0, err
		}
		d.Status = DeliveryStatus(statusStr)
		list = append(list, &d)
	}

	return list, total, rows.Err()
}

func (s *SQLiteStorage) GetPendingDeliveries(ctx context.Context, limit int) ([]*DeliveryWithDetails, error) {
	if limit <= 0 {
		limit = 50
	}
	// Only fetch orphaned pending deliveries older than 5 seconds to prevent race with in-memory queue
	orphanThreshold := time.Now().UTC().Add(-5 * time.Second)
	query := `
		SELECT d.id, d.message_id, d.endpoint_id, d.status, d.attempt_number, d.max_attempts,
		       d.http_status, d.duration_ms, d.request_headers, d.response_headers,
		       d.response_body, d.error_message, d.next_retry_at, d.sent_at, d.created_at,
		       e.name, e.url, m.event_type, m.payload
		FROM deliveries d
		JOIN endpoints e ON d.endpoint_id = e.id
		JOIN messages m ON d.message_id = m.id
		WHERE d.status = 'PENDING' AND d.created_at <= ?
		ORDER BY d.created_at ASC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, orphanThreshold, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeliveryWithDetails
	for rows.Next() {
		var d DeliveryWithDetails
		var statusStr string
		err := rows.Scan(
			&d.ID, &d.MessageID, &d.EndpointID, &statusStr, &d.AttemptNumber, &d.MaxAttempts,
			&d.HTTPStatus, &d.DurationMs, &d.RequestHeaders, &d.ResponseHeaders,
			&d.ResponseBody, &d.ErrorMessage, &d.NextRetryAt, &d.SentAt, &d.CreatedAt,
			&d.EndpointName, &d.EndpointURL, &d.EventType, &d.Payload,
		)
		if err != nil {
			return nil, err
		}
		d.Status = DeliveryStatus(statusStr)
		list = append(list, &d)
	}
	return list, rows.Err()
}

func (s *SQLiteStorage) GetDueRetries(ctx context.Context, limit int) ([]*DeliveryWithDetails, error) {
	if limit <= 0 {
		limit = 50
	}
	now := time.Now().UTC()
	query := `
		SELECT d.id, d.message_id, d.endpoint_id, d.status, d.attempt_number, d.max_attempts,
		       d.http_status, d.duration_ms, d.request_headers, d.response_headers,
		       d.response_body, d.error_message, d.next_retry_at, d.sent_at, d.created_at,
		       e.name, e.url, m.event_type, m.payload
		FROM deliveries d
		JOIN endpoints e ON d.endpoint_id = e.id
		JOIN messages m ON d.message_id = m.id
		WHERE d.status = 'RETRY' AND d.next_retry_at <= ?
		ORDER BY d.next_retry_at ASC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, now, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeliveryWithDetails
	for rows.Next() {
		var d DeliveryWithDetails
		var statusStr string
		err := rows.Scan(
			&d.ID, &d.MessageID, &d.EndpointID, &statusStr, &d.AttemptNumber, &d.MaxAttempts,
			&d.HTTPStatus, &d.DurationMs, &d.RequestHeaders, &d.ResponseHeaders,
			&d.ResponseBody, &d.ErrorMessage, &d.NextRetryAt, &d.SentAt, &d.CreatedAt,
			&d.EndpointName, &d.EndpointURL, &d.EventType, &d.Payload,
		)
		if err != nil {
			return nil, err
		}
		d.Status = DeliveryStatus(statusStr)
		list = append(list, &d)
	}
	return list, rows.Err()
}

func (s *SQLiteStorage) ReplayDelivery(ctx context.Context, deliveryID string) (*Delivery, error) {
	d, err := s.GetDelivery(ctx, deliveryID)
	if err != nil {
		return nil, err
	}

	newD := &Delivery{
		ID:            "del_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16],
		MessageID:     d.MessageID,
		EndpointID:    d.EndpointID,
		Status:        DeliveryStatusPending,
		AttemptNumber: 0,
		MaxAttempts:   d.MaxAttempts,
		CreatedAt:     time.Now().UTC(),
	}

	if err := s.CreateDelivery(ctx, newD); err != nil {
		return nil, err
	}
	return newD, nil
}

func (s *SQLiteStorage) ReplayAllDLQ(ctx context.Context) (int, error) {
	query := `
		SELECT id, message_id, endpoint_id, max_attempts
		FROM deliveries
		WHERE status = 'FAILED'
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type dlqItem struct {
		messageID   string
		endpointID  string
		maxAttempts int
	}
	var items []dlqItem
	for rows.Next() {
		var id, msgID, epID string
		var maxAtt int
		if err := rows.Scan(&id, &msgID, &epID, &maxAtt); err != nil {
			return 0, err
		}
		items = append(items, dlqItem{messageID: msgID, endpointID: epID, maxAttempts: maxAtt})
	}

	replayed := 0
	for _, it := range items {
		newD := &Delivery{
			ID:            "del_" + strings.ReplaceAll(uuid.New().String(), "-", "")[:16],
			MessageID:     it.messageID,
			EndpointID:    it.endpointID,
			Status:        DeliveryStatusPending,
			AttemptNumber: 0,
			MaxAttempts:   it.maxAttempts,
			CreatedAt:     time.Now().UTC(),
		}
		if err := s.CreateDelivery(ctx, newD); err == nil {
			replayed++
		}
	}

	return replayed, nil
}

// =========================================================================
// Metrics
// =========================================================================

func (s *SQLiteStorage) GetStats(ctx context.Context) (*StatsSummary, error) {
	stats := &StatsSummary{}

	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages").Scan(&stats.TotalMessages)

	query := `
		SELECT status, COUNT(*), COALESCE(AVG(duration_ms), 0)
		FROM deliveries
		GROUP BY status
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err == nil {
		var totalDurSum float64
		var countWithDur int64

		for rows.Next() {
			var st string
			var cnt int64
			var avgDur float64
			if err := rows.Scan(&st, &cnt, &avgDur); err == nil {
				stats.TotalDeliveries += cnt
				switch DeliveryStatus(st) {
				case DeliveryStatusSuccess:
					stats.SuccessfulCount = cnt
					totalDurSum += avgDur * float64(cnt)
					countWithDur += cnt
				case DeliveryStatusRetry:
					stats.RetryingCount = cnt
				case DeliveryStatusFailed:
					stats.FailedCount = cnt
				case DeliveryStatusPending, DeliveryStatusInFlight:
					stats.PendingCount += cnt
				}
			}
		}
		rows.Close()

		if stats.TotalDeliveries > 0 {
			stats.SuccessRate = float64(stats.SuccessfulCount) / float64(stats.TotalDeliveries) * 100.0
		}
		if countWithDur > 0 {
			stats.AvgDurationMs = totalDurSum / float64(countWithDur)
		}
	}

	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM endpoints WHERE status = 'ACTIVE'").Scan(&stats.ActiveEndpoints)
	_ = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM endpoints WHERE status = 'SUSPENDED'").Scan(&stats.SuspendedEndpoints)

	return stats, nil
}
