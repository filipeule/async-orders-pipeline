package worker

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/filipeule/integration-pipeline/internal/domain"
	"github.com/filipeule/integration-pipeline/internal/queue"
	"github.com/filipeule/integration-pipeline/internal/repository"
)

var leadBackoffs = []time.Duration{time.Second, 4 * time.Second, 16 * time.Second}

func RunLeadConsumer(ctx context.Context, store repository.Storage, pub *queue.Publisher, conn *queue.Connection) {
	log := slog.With("worker", "lead_consumer")

	msgs, err := conn.NewConsumer(queue.QueueLeadReceived)
	if err != nil {
		log.Error("consumer_setup_failed", "error", err)
		return
	}
	log.Info("started")

	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				log.Warn("channel_closed")
				return
			}

			var msg domain.LeadMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("unmarshal_failed", "error", err)
				d.Ack(false)
				continue
			}

			log := log.With("correlation_id", msg.CorrelationID, "gateway", msg.Gateway,
				"transaction_id", msg.Payload.TransactionID)

			var processErr error
			for attempt := 0; attempt <= len(leadBackoffs); attempt++ {
				if attempt > 0 {
					time.Sleep(leadBackoffs[attempt-1])
					log.Info("retry", "attempt", attempt)
				}
				processErr = processLead(ctx, store, pub, &msg)
				if processErr == nil {
					break
				}
				log.Warn("process_error", "attempt", attempt, "error", processErr)
			}

			if processErr != nil {
				log.Error("sent_to_dlq", "reason", processErr.Error())
				pub.Publish(ctx, queue.QueueDLQConsumer, domain.DLQMessage{
					CorrelationID: msg.CorrelationID,
					Gateway:       msg.Gateway,
					OriginalBody:  msg.Payload,
					Reason:        processErr.Error(),
				})
				store.InsertDeadLetter(ctx, msg.CorrelationID, "consumer", processErr.Error(), msg.Payload)
			}

			d.Ack(false)
		}
	}
}

func processLead(ctx context.Context, store repository.Storage, pub *queue.Publisher, msg *domain.LeadMessage) error {
	orderID, err := store.UpsertLeadAndOrder(ctx, msg.CorrelationID, msg.Gateway, msg.Payload)
	if err != nil {
		return err
	}

	channels := []struct{ name, q string }{
		{"SMS", queue.QueueDistSMS},
		{"EMAIL", queue.QueueDistEmail},
		{"CALL_CENTER", queue.QueueDistCallCenter},
		{"WHATSAPP", queue.QueueDistWhatsApp},
	}
	for _, ch := range channels {
		if err := pub.Publish(ctx, ch.q, domain.DistMessage{
			CorrelationID: msg.CorrelationID,
			OrderID:       orderID,
			Gateway:       msg.Gateway,
			Channel:       ch.name,
			CreatedAt:     time.Now().UTC(),
		}); err != nil {
			return err
		}
	}

	slog.Info("lead_persisted",
		"correlation_id", msg.CorrelationID,
		"order_id", orderID,
		"event", msg.Payload.Event,
	)
	return nil
}
