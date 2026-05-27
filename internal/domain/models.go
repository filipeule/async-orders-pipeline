package domain

import "time"

type WebhookPayload struct {
	TransactionID   string    `json:"transaction_id"`
	TransactionTime time.Time `json:"transaction_time"`
	Event           string    `json:"event"`
	Customer        Customer  `json:"customer"`
	Product         Product   `json:"product"`
	Payment         Payment   `json:"payment"`
}

type Customer struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Phone     string `json:"phone"`
	Country   string `json:"country"`
}

type Product struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Niche    string `json:"niche"`
	Quantity int    `json:"quantity"`
}

type Payment struct {
	Status    string  `json:"status"`
	AmountUSD float64 `json:"amount_usd"`
	Method    string  `json:"method"`
}

// GurenEncrypted is the JSON envelope guren sends when the body is
// AES-256-GCM encrypted. Ciphertext includes the 16-byte auth tag at the end
// (Go's AEAD.Seal output format).
type GurenEncrypted struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

// LeadMessage is published to lead.received queue.
type LeadMessage struct {
	CorrelationID string          `json:"correlation_id"`
	Gateway       string          `json:"gateway"`
	ReceivedAt    time.Time       `json:"received_at"`
	Payload       *WebhookPayload `json:"payload"`
	PhoneValid    bool            `json:"phone_valid"`
}

// DistMessage is published to dist.* queues.
type DistMessage struct {
	CorrelationID string    `json:"correlation_id"`
	OrderID       int64     `json:"order_id"`
	Gateway       string    `json:"gateway"`
	Channel       string    `json:"channel"`
	CreatedAt     time.Time `json:"created_at"`
}

// DLQMessage is published to any dead-letter queue.
type DLQMessage struct {
	CorrelationID string `json:"correlation_id"`
	Gateway       string `json:"gateway"`
	OriginalBody  any    `json:"original_body"`
	Reason        string `json:"reason"`
}
