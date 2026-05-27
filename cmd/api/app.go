package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/filipeule/integration-pipeline/internal/config"
	"github.com/filipeule/integration-pipeline/internal/crypto"
	"github.com/filipeule/integration-pipeline/internal/domain"
	"github.com/filipeule/integration-pipeline/internal/queue"
	"github.com/filipeule/integration-pipeline/internal/repository"
)

type application struct {
	port      string
	storage   repository.Storage
	config    *config.Config
	publisher *queue.Publisher
}

func (app *application) decryptGuren(rawBytes []byte) ([]byte, domain.WebhookPayload, error) {
	var enc domain.GurenEncrypted
	var payload domain.WebhookPayload
	if err := json.Unmarshal(rawBytes, &enc); err != nil || enc.Nonce == "" || enc.Ciphertext == "" {
		return nil, payload, fmt.Errorf("malformed encrypted envelope")
	}

	decryptedBytes, err := crypto.DecryptAES256GCM(app.config.GurenKey, enc.Nonce, enc.Ciphertext)
	if err != nil {
		return nil, payload, err
	}

	if err := json.Unmarshal(decryptedBytes, &payload); err != nil {
		return decryptedBytes, payload, fmt.Errorf("decrypted payload is not valid json: %w", err)
	}

	return decryptedBytes, payload, nil
}

func (app *application) routeToDLQ(
	ctx context.Context,
	corrID, gateway string,
	originalBody any,
	queueName, origin, reason string,
	log *slog.Logger,
) {
	log.Warn("dlq", "queue", queueName, "reason", reason)
	app.publisher.Publish(ctx, queueName, domain.DLQMessage{
		CorrelationID: corrID,
		Gateway:       gateway,
		OriginalBody:  originalBody,
		Reason:        reason,
	})
	app.storage.InsertDeadLetter(ctx, corrID, origin, reason, originalBody)
}