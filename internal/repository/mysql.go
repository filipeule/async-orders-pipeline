package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/filipeule/integration-pipeline/internal/domain"
)

type MySQLStorage struct {
	db *sql.DB
}

func NewMySQLStorage(db *sql.DB) Storage {
	return &MySQLStorage{db: db}
}

func (s *MySQLStorage) InsertRawPayload(ctx context.Context, correlationID, gateway string, headers map[string]string, bodyRaw []byte, bodyDecrypted []byte) error {
	headersJSON, _ := json.Marshal(headers)
	var decrypted sql.NullString
	if bodyDecrypted != nil {
		decrypted = sql.NullString{String: string(bodyDecrypted), Valid: true}
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO raw_payloads (correlation_id, gateway, received_at, headers, body_raw, body_decrypted)
		 VALUES (?, ?, UTC_TIMESTAMP(), ?, ?, ?)`,
		correlationID, gateway, string(headersJSON), string(bodyRaw), decrypted,
	)
	return err
}

func (s *MySQLStorage) UpsertLeadAndOrder(ctx context.Context, correlationID, gateway string, p *domain.WebhookPayload) (int64, error) {
	_, err := s.db.ExecContext(ctx,
		`CALL sp_insert_lead(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, @order_id)`,
		correlationID,
		gateway,
		p.Customer.Email,
		p.Customer.FirstName,
		p.Customer.LastName,
		p.Customer.Phone,
		p.Customer.Country,
		p.TransactionID,
		p.TransactionTime,
		p.Product.ID,
		p.Product.Name,
		p.Product.Niche,
		p.Product.Quantity,
		p.Payment.AmountUSD,
		p.Payment.Method,
		p.Payment.Status,
		p.Event,
	)
	if err != nil {
		return 0, err
	}

	var orderID int64
	err = s.db.QueryRowContext(ctx, `SELECT @order_id`).Scan(&orderID)
	return orderID, err
}

func (s *MySQLStorage) MarkDelivered(ctx context.Context, orderID int64, channel string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx,
		`UPDATE distribution_status
		 SET status = 'delivered',
		     delivered_at = ?,
		     lag_db_channel_seconds = TIMESTAMPDIFF(SECOND, created_at, ?)
		 WHERE order_id = ? AND channel = ?`,
		now, now, orderID, channel,
	)
	return err
}

func (s *MySQLStorage) MarkFailed(ctx context.Context, orderID int64, channel string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE distribution_status SET status = 'failed' WHERE order_id = ? AND channel = ?`,
		orderID, channel,
	)
	return err
}

func (s *MySQLStorage) InsertDeadLetter(ctx context.Context, correlationID, origin, reason string, payload any) error {
	b, _ := json.Marshal(payload)
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO lead_dead_letter (correlation_id, origin, payload, error_reason) VALUES (?, ?, ?, ?)`,
		correlationID, origin, string(b), reason,
	)
	return err
}

func (s *MySQLStorage) ClaimIdempotencyKey(ctx context.Context, transactionID, event, gateway, correlationID string) (bool, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT IGNORE INTO idempotency_keys (transaction_id, event, gateway, correlation_id)
		 VALUES (?, ?, ?, ?)`,
		transactionID, event, gateway, correlationID,
	)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *MySQLStorage) ReleaseIdempotencyKey(ctx context.Context, transactionID, event string) error {
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM idempotency_keys WHERE transaction_id = ? AND event = ?`,
		transactionID, event,
	)
	return err
}
