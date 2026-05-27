package repository

import (
	"context"

	"github.com/filipeule/integration-pipeline/internal/domain"
)

type Storage interface {
	InsertRawPayload(
		ctx context.Context,
		correlationID, gateway string,
		headers map[string]string,
		bodyRaw []byte,
		bodyDecrypted []byte,
	) error
	UpsertLeadAndOrder(
		ctx context.Context, correlationID, gateway string, p *domain.WebhookPayload,
	) (int64, error)
	MarkDelivered(ctx context.Context, orderID int64, channel string) error
	MarkFailed(ctx context.Context, orderID int64, channel string) error
	InsertDeadLetter(ctx context.Context, correlationID, origin, reason string, payload any) error
	ClaimIdempotencyKey(ctx context.Context, transactionID, event, gateway, correlationID string) (bool, error)
	ReleaseIdempotencyKey(ctx context.Context, transactionID, event string) error
}