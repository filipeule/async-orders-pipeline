package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/filipeule/integration-pipeline/internal/domain"
	"github.com/filipeule/integration-pipeline/internal/queue"
	"github.com/filipeule/integration-pipeline/internal/service"
	"github.com/google/uuid"
)

func (app *application) webhookHandler(w http.ResponseWriter, r *http.Request) {
	gateway := r.PathValue("gateway")
	if gateway != "guren" && gateway != "lancelot" {
		app.jsonRes(w, http.StatusNotFound, map[string]string{"error": "unknown gateway"})
		return
	}

	uuidv7, err := uuid.NewV7()
	if err != nil {
		app.jsonRes(
			w, http.StatusInternalServerError, map[string]string{"error": "failed to generate correlation_id"},
		)
		return
	}
	corrID := uuidv7.String()

	ctx := r.Context()
	log := slog.With("correlation_id", corrID, "gateway", gateway)

	headers := make(map[string]string, len(r.Header))
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}

	rawBytes, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		log.Error("read_body_failed", "error", err.Error())
		app.jsonRes(w, http.StatusBadRequest, map[string]string{"error": "cannot read body"})
		return
	}

	var rawBody map[string]any
	if err := json.Unmarshal(rawBytes, &rawBody); err != nil {
		log.Error("unmarshal_body_failed", "error", err.Error())
		app.jsonRes(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}

	var payload domain.WebhookPayload
	var decryptedBytes []byte

	if gateway == "guren" && r.Header.Get("X-Guren-Encrypted") == "true" {
		decryptedBytes, payload, err = app.decryptGuren(rawBytes)
		if err != nil {
			app.storage.InsertRawPayload(ctx, corrID, gateway, headers, rawBytes, decryptedBytes)
			if decryptedBytes != nil {
				app.routeToDLQ(ctx, corrID, gateway, rawBody, queue.QueueDLQSchema, "schema_validation", err.Error(), log)
			} else {
				app.routeToDLQ(ctx, corrID, gateway, rawBody, queue.QueueDLQDecrypt, "decrypt", err.Error(), log)
			}
			app.jsonRes(w, http.StatusBadRequest, map[string]string{"status": "dlq"})
			return
		}
	} else {
		if err := json.Unmarshal(rawBytes, &payload); err != nil {
			app.storage.InsertRawPayload(ctx, corrID, gateway, headers, rawBytes, nil)
			app.routeToDLQ(ctx, corrID, gateway, rawBody, queue.QueueDLQSchema, "schema_validation", "cannot parse payload: "+err.Error(), log)
			app.jsonRes(w, http.StatusBadRequest, map[string]string{"status": "dlq"})
			return
		}
	}

	err = app.storage.InsertRawPayload(ctx, corrID, gateway, headers, rawBytes, decryptedBytes)
	if err != nil {
		log.Error("failed_to_insert", "error", err.Error())
		app.jsonRes(w, http.StatusInternalServerError, map[string]string{"status": "internal error"})
		return
	}

	service.Normalize(&payload)
	result := service.Validate(&payload)
	if !result.Valid {
		reason := "invalid fields: " + strings.Join(result.MissingFields, ", ")
		log.Info("schema_failed", "reason", reason)
		app.routeToDLQ(ctx, corrID, gateway, rawBody, queue.QueueDLQSchema, "schema_validation", reason, log)
		app.jsonRes(w, http.StatusBadRequest, map[string]string{"status": "dlq"})
		return
	}

	if payload.Event != "order.approved" || payload.Payment.Status != "approved" {
		log.Info("discarded", "event", payload.Event, "payment_status", payload.Payment.Status)
		app.jsonRes(w, http.StatusOK, map[string]string{"status": "discarded"})
		return
	}

	claimed, err := app.storage.ClaimIdempotencyKey(ctx, payload.TransactionID, payload.Event, gateway, corrID)
	if err != nil {
		log.Error("idempotency_claim_failed", "error", err)
		app.jsonRes(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if !claimed {
		log.Info("duplicate", "transaction_id", payload.TransactionID)
		app.jsonRes(w, http.StatusConflict, map[string]string{"status": "duplicate"})
		return
	}

	msg := domain.LeadMessage{
		CorrelationID: corrID,
		Gateway:       gateway,
		ReceivedAt:    time.Now().UTC(),
		Payload:       &payload,
		PhoneValid:    result.PhoneValid,
	}
	if err := app.publisher.Publish(ctx, queue.QueueLeadReceived, msg); err != nil {
		log.Error("publish_failed", "error", err)
		app.storage.ReleaseIdempotencyKey(ctx, payload.TransactionID, payload.Event)
		app.jsonRes(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	log.Info("accepted", "transaction_id", payload.TransactionID, "event", payload.Event)
	app.jsonRes(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (app *application) healthHandler(w http.ResponseWriter, r *http.Request) {
	app.jsonRes(w, http.StatusOK, map[string]string{"status": "ok"})
}
