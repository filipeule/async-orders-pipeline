// Generates webhook_payloads_v2.json with 20 payloads themed on Code Geass
// characters. Encrypts guren payloads with AES-256-GCM using the key from
// docker-compose.yml. Run from the project root:
//
//	go run ./cmd/script/generate
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

const (
	gurenKey = "3b65ec7a99cd009b6e89ee0eed8cf660beb7c35bfd2add7e09e9232be02a2839"
	outFile  = "data/webhook_payloads.json"
)

type Payload struct {
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

type Encrypted struct {
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

type WebhookEntry struct {
	Gateway string            `json:"gateway"`
	Headers map[string]string `json:"headers"`
	Body    json.RawMessage   `json:"body"`
}

var gcm cipher.AEAD

func init() {
	keyBytes, err := hex.DecodeString(gurenKey)
	if err != nil {
		log.Fatalf("invalid key hex: %v", err)
	}
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		log.Fatalf("aes init: %v", err)
	}
	g, err := cipher.NewGCM(block)
	if err != nil {
		log.Fatalf("gcm init: %v", err)
	}
	gcm = g
}

func encrypt(p Payload) Encrypted {
	plaintext, _ := json.Marshal(p)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		log.Fatalf("nonce gen: %v", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return Encrypted{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
	}
}

// encryptCorrupt encrypts then flips a byte in the ciphertext, so GCM tag
// verification fails when decrypted
func encryptCorrupt(p Payload) Encrypted {
	enc := encrypt(p)
	ct, _ := base64.StdEncoding.DecodeString(enc.Ciphertext)
	if len(ct) > 5 {
		ct[3] ^= 0xFF
	}
	enc.Ciphertext = base64.StdEncoding.EncodeToString(ct)
	return enc
}

func lancelot(p Payload) WebhookEntry {
	body, _ := json.Marshal(p)
	return WebhookEntry{
		Gateway: "lancelot",
		Headers: map[string]string{"Content-Type": "application/json"},
		Body:    body,
	}
}

func guren(p Payload) WebhookEntry {
	body, _ := json.Marshal(encrypt(p))
	return WebhookEntry{
		Gateway: "guren",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Guren-Encrypted": "true",
		},
		Body: body,
	}
}

func gurenCorrupted(p Payload) WebhookEntry {
	body, _ := json.Marshal(encryptCorrupt(p))
	return WebhookEntry{
		Gateway: "guren",
		Headers: map[string]string{
			"Content-Type":      "application/json",
			"X-Guren-Encrypted": "true",
		},
		Body: body,
	}
}

func tt(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

func main() {
	// ===================== LANCELOT (plain JSON) =====================

	lelouch := Payload{
		TransactionID:   "ORD-2026-001",
		TransactionTime: tt("2026-05-20T10:00:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "lelouch.lamperouge@codegeass.com", FirstName: "Lelouch",
			LastName: "Lamperouge", Phone: "+5511910001001", Country: "BR",
		},
		Product: Product{ID: "PROD-001", Name: "Zero Mask", Niche: "rebellion_supplies", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 199.99, Method: "credit_card"},
	}

	suzaku := Payload{
		TransactionID:   "ORD-2026-002",
		TransactionTime: tt("2026-05-20T11:15:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "suzaku.kururugi@codegeass.com", FirstName: "Suzaku",
			LastName: "Kururugi", Phone: "+819011002002", Country: "JP",
		},
		Product: Product{ID: "PROD-002", Name: "Knight of Zero Insignia", Niche: "knightmare_combat", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 599.00, Method: "credit_card"},
	}

	kallen := Payload{
		TransactionID:   "ORD-2026-003",
		TransactionTime: tt("2026-05-20T12:30:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "kallen.kozuki@codegeass.com", FirstName: "Kallen",
			LastName: "Kozuki", Phone: "+819011003003", Country: "JP",
		},
		Product: Product{ID: "PROD-003", Name: "Guren Hex Beam Module", Niche: "rebellion_supplies", Quantity: 2},
		Payment: Payment{Status: "approved", AmountUSD: 84.50, Method: "paypal"},
	}

	cc := Payload{
		TransactionID:   "ORD-2026-004",
		TransactionTime: tt("2026-05-20T13:45:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "cc@codegeass.com", FirstName: "C", LastName: "C",
			Phone: "+5511910004004", Country: "BR",
		},
		Product: Product{ID: "PROD-004", Name: "Pizza Hut Voucher Pack", Niche: "gourmet", Quantity: 5},
		Payment: Payment{Status: "approved", AmountUSD: 75.00, Method: "credit_card"},
	}

	shirley := Payload{
		TransactionID:   "ORD-2026-008",
		TransactionTime: tt("2026-05-20T15:20:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "shirley-fenette@codegeass.com", FirstName: "Shirley",
			LastName: "Fenette", Phone: "+5511910008008", Country: "BR",
		},
		Product: Product{ID: "PROD-008", Name: "Tennis Racket", Niche: "ashford_academy", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 65.00, Method: "credit_card"},
	}

	// Validation fail (bad email)
	euphemia := Payload{
		TransactionID:   "ORD-2026-005",
		TransactionTime: tt("2026-05-20T14:00:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "euphemia.libritannia-not-an-email", FirstName: "Euphemia",
			LastName: "li Britannia", Phone: "+5511910005005", Country: "BR",
		},
		Product: Product{ID: "PROD-005", Name: "SAZ Membership Kit", Niche: "charity", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 150.00, Method: "paypal"},
	}

	// Validation fail (empty transaction_id)
	nunnally := Payload{
		TransactionID:   "",
		TransactionTime: tt("2026-05-20T16:00:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "nunnally.lamperouge@codegeass.com", FirstName: "Nunnally",
			LastName: "Lamperouge", Phone: "+5511910009009", Country: "BR",
		},
		Product: Product{ID: "PROD-009", Name: "Origami Crane Set", Niche: "peace_advocacy", Quantity: 10},
		Payment: Payment{Status: "approved", AmountUSD: 35.00, Method: "credit_card"},
	}

	// Non-approved (refunded)
	milly := Payload{
		TransactionID:   "ORD-2026-010",
		TransactionTime: tt("2026-05-20T17:30:00Z"),
		Event:           "order.refunded",
		Customer: Customer{
			Email: "milly.ashford@codegeass.com", FirstName: "Milly",
			LastName: "Ashford", Phone: "+5511910010010", Country: "BR",
		},
		Product: Product{ID: "PROD-010", Name: "Cosplay Wedding Dress", Niche: "ashford_academy", Quantity: 1},
		Payment: Payment{Status: "refunded", AmountUSD: 320.00, Method: "credit_card"},
	}

	// ===================== GUREN (encrypted) =====================

	schneizel := Payload{
		TransactionID:   "ORD-2026-011",
		TransactionTime: tt("2026-05-20T10:30:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "schneizel.elbritannia@codegeass.com", FirstName: "Schneizel",
			LastName: "el Britannia", Phone: "+5511910011011", Country: "BR",
		},
		Product: Product{ID: "PROD-011", Name: "Damocles Access Card", Niche: "imperial_goods", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 9999.99, Method: "credit_card"},
	}

	cornelia := Payload{
		TransactionID:   "ORD-2026-012",
		TransactionTime: tt("2026-05-20T11:00:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "cornelia.libritannia@codegeass.com", FirstName: "Cornelia",
			LastName: "li Britannia", Phone: "+5511910012012", Country: "BR",
		},
		Product: Product{ID: "PROD-012", Name: "Gloucester Combat Manual", Niche: "knightmare_combat", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 749.00, Method: "credit_card"},
	}

	tohdoh := Payload{
		TransactionID:   "ORD-2026-013",
		TransactionTime: tt("2026-05-20T12:15:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "tohdoh.kyoshiro@codegeass.com", FirstName: "Kyoshiro",
			LastName: "Tohdoh", Phone: "+819011013013", Country: "JP",
		},
		Product: Product{ID: "PROD-013", Name: "Zen Garden Set", Niche: "meditation", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 120.00, Method: "paypal"},
	}

	lloyd := Payload{
		TransactionID:   "ORD-2026-014",
		TransactionTime: tt("2026-05-20T13:30:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "lloyd.asplund@codegeass.com", FirstName: "Lloyd",
			LastName: "Asplund", Phone: "+5511910014014", Country: "BR",
		},
		Product: Product{ID: "PROD-014", Name: "Engineer Lab Coat + Goggles", Niche: "knightmare_research", Quantity: 2},
		Payment: Payment{Status: "approved", AmountUSD: 89.50, Method: "credit_card"},
	}

	jeremiah := Payload{
		TransactionID:   "ORD-2026-015",
		TransactionTime: tt("2026-05-20T14:45:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "jeremiah.gottwald@codegeass.com", FirstName: "Jeremiah",
			LastName: "Gottwald", Phone: "+5511910015015", Country: "BR",
		},
		Product: Product{ID: "PROD-015", Name: "Orange Plantation Starter Kit", Niche: "gardening", Quantity: 3},
		Payment: Payment{Status: "approved", AmountUSD: 45.00, Method: "credit_card"},
	}

	// Validation fail (bad email)
	rolo := Payload{
		TransactionID:   "ORD-2026-018",
		TransactionTime: tt("2026-05-20T15:30:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "rolo-no-domain", FirstName: "Rolo",
			LastName: "Lamperouge", Phone: "+5511910018018", Country: "BR",
		},
		Product: Product{ID: "PROD-018", Name: "Geass Stopwatch", Niche: "supernatural", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 299.00, Method: "credit_card"},
	}

	// Decrypt fail (corrupted ciphertext)
	anya := Payload{
		TransactionID:   "ORD-2026-019",
		TransactionTime: tt("2026-05-20T16:15:00Z"),
		Event:           "order.approved",
		Customer: Customer{
			Email: "anya.alstreim@codegeass.com", FirstName: "Anya",
			LastName: "Alstreim", Phone: "+5511910019019", Country: "BR",
		},
		Product: Product{ID: "PROD-019", Name: "Diary Camera", Niche: "photography", Quantity: 1},
		Payment: Payment{Status: "approved", AmountUSD: 220.00, Method: "credit_card"},
	}

	// Non-approved (declined)
	gino := Payload{
		TransactionID:   "ORD-2026-020",
		TransactionTime: tt("2026-05-20T17:00:00Z"),
		Event:           "order.declined",
		Customer: Customer{
			Email: "gino.weinberg@codegeass.com", FirstName: "Gino",
			LastName: "Weinberg", Phone: "+5511910020020", Country: "BR",
		},
		Product: Product{ID: "PROD-020", Name: "Tristan Knight Helmet", Niche: "knightmare_combat", Quantity: 1},
		Payment: Payment{Status: "declined", AmountUSD: 480.00, Method: "credit_card"},
	}

	// ===================== Assemble =====================
	entries := []WebhookEntry{
		// 10 OK alternados
		lancelot(lelouch), guren(schneizel),
		lancelot(suzaku), guren(cornelia),
		lancelot(kallen), guren(tohdoh),
		lancelot(cc), guren(lloyd),
		lancelot(euphemia), guren(jeremiah),

		// 4 duplicatas (lancelot byte-a-byte, guren mesma plaintext com nonce novo)
		lancelot(lelouch),
		guren(schneizel),
		lancelot(kallen),
		guren(tohdoh),

		// 3 validation fails
		lancelot(shirley),
		guren(rolo),
		lancelot(nunnally),

		// 1 decrypt fail
		gurenCorrupted(anya),

		// 2 non-approved
		lancelot(milly),
		guren(gino),
	}

	out, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		log.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(outFile, out, 0644); err != nil {
		log.Fatalf("write: %v", err)
	}

	fmt.Printf("wrote %d entries to %s\n", len(entries), outFile)
	fmt.Printf("\nexpected distribution after running test-webhooks:\n")
	fmt.Printf("  valid_approved_unique_in_lead_events  10\n")
	fmt.Printf("  duplicates_natural_key                 4\n")
	fmt.Printf("  dlq_decrypt_failed                     1\n")
	fmt.Printf("  dlq_schema_failed                      3\n")
	fmt.Printf("  non_approved_discarded                 2\n")
}
