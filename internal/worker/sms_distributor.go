package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"github.com/filipeule/integration-pipeline/internal/domain"
	"github.com/filipeule/integration-pipeline/internal/queue"
	"github.com/filipeule/integration-pipeline/internal/repository"
)

var smsBackoffs = []time.Duration{time.Second, 4 * time.Second, 16 * time.Second}

// Seed rand once at package init
func init() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
}

func RunSMSDistributor(ctx context.Context, store repository.Storage, pub *queue.Publisher, conn *queue.Connection, webhookURL string) {
	log := slog.With("worker", "sms_distributor")

	msgs, err := conn.NewConsumer(queue.QueueDistSMS)
	if err != nil {
		log.Error("consumer_setup_failed", "error", err)
		return
	}
	log.Info("started", "webhook_url", webhookURL)

	for {
		select {
		case <-ctx.Done():
			return
		case d, ok := <-msgs:
			if !ok {
				log.Warn("channel_closed")
				return
			}

			var msg domain.DistMessage
			if err := json.Unmarshal(d.Body, &msg); err != nil {
				log.Error("unmarshal_failed", "error", err)
				d.Ack(false)
				continue
			}

			log := log.With("correlation_id", msg.CorrelationID, "order_id", msg.OrderID)

			var lastErr error
			for attempt := 0; attempt <= len(smsBackoffs); attempt++ {
				if attempt > 0 {
					time.Sleep(smsBackoffs[attempt-1])
					log.Info("retry", "attempt", attempt)
				}
				lastErr = sendSMS(ctx, webhookURL, &msg)
				if lastErr == nil {
					break
				}
				log.Warn("send_error", "attempt", attempt, "error", lastErr)
			}

			if lastErr != nil {
				log.Error("sent_to_dlq", "reason", lastErr.Error())
				pub.Publish(ctx, queue.QueueDLQSMS, domain.DLQMessage{
					CorrelationID: msg.CorrelationID,
					Gateway:       msg.Gateway,
					OriginalBody:  msg,
					Reason:        lastErr.Error(),
				})
				store.InsertDeadLetter(ctx, msg.CorrelationID, "sms_distributor", lastErr.Error(), msg)
				store.MarkFailed(ctx, msg.OrderID, "SMS")
			} else {
				store.MarkDelivered(ctx, msg.OrderID, "SMS")
				log.Info("sms_delivered", "order_id", msg.OrderID)
			}

			d.Ack(false)
		}
	}
}

func sendSMS(ctx context.Context, webhookURL string, msg *domain.DistMessage) error {
	// simulate 10% random failure per attempt
	if rand.Float64() < 0.10 {
		return fmt.Errorf("simulated failure (10%% random)")
	}

	if webhookURL == "" {
		return nil // no URL configured, skip http call
	}

	b, _ := json.Marshal(msg)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-Id", msg.CorrelationID)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook.site returned %d", resp.StatusCode)
	}
	return nil
}
