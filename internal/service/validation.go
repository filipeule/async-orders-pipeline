package service

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/filipeule/integration-pipeline/internal/domain"
)

var (
	emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

type ValidationResult struct {
	Valid         bool
	MissingFields []string
	PhoneValid    bool
}

func Normalize(p *domain.WebhookPayload) {
    p.Customer.Email = strings.ToLower(strings.TrimSpace(p.Customer.Email))
    p.Customer.Phone = normalizePhone(p.Customer.Phone)
    p.Customer.FirstName = strings.TrimSpace(p.Customer.FirstName)
    if p.Customer.FirstName == "" {
        p.Customer.FirstName = "Customer"
    }
    p.Customer.LastName = strings.TrimSpace(p.Customer.LastName)
}

func Validate(p *domain.WebhookPayload) ValidationResult {
	var missing []string

	if p.TransactionID == "" {
		missing = append(missing, "transaction_id")
	}

	if p.Event == "" {
		missing = append(missing, "event")
	}

	if p.Customer.Country == "" {
		missing = append(missing, "customer.country")
	}

	if p.Product.ID == "" {
		missing = append(missing, "product.id")
	}

	if p.Payment.Status == "" {
		missing = append(missing, "payment.status")
	}

	if p.Customer.Email == "" || !emailRe.MatchString(p.Customer.Email) {
		missing = append(missing, "customer.email")
	}

	phoneValid := len(p.Customer.Phone) >= 10

	return ValidationResult{
		Valid:         len(missing) == 0,
		MissingFields: missing,
		PhoneValid:    phoneValid,
	}
}

func normalizePhone(phone string) string {
	hasPlus := strings.HasPrefix(strings.TrimSpace(phone), "+")
	digits := strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)
	if hasPlus {
		return "+" + digits
	}
	return digits
}
