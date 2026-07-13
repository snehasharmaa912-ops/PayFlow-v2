package payflow

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

type Charge struct {
	ID             string    `json:"id"`
	Amount         int64     `json:"amount"`
	Currency       string    `json:"currency"`
	CustomerID     string    `json:"customer_id"`
	IdempotencyKey string    `json:"idempotency_key"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

type CreateChargeRequest struct {
	Amount         int64  `json:"amount"`
	Currency       string `json:"currency"`
	CustomerID     string `json:"customer_id"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func validateCreateChargeRequest(req CreateChargeRequest) *ValidationError {
	if req.Amount <= 0 {
		return &ValidationError{Field: "amount", Message: "must be a positive integer number of cents"}
	}

	currency := strings.TrimSpace(req.Currency)
	if len(currency) != 3 {
		return &ValidationError{Field: "currency", Message: "must be a 3-letter ISO currency code, e.g. USD"}
	}
	for _, r := range currency {
		isUpper := r >= 'A' && r <= 'Z'
		isLower := r >= 'a' && r <= 'z'
		if !isUpper && !isLower {
			return &ValidationError{Field: "currency", Message: "must contain only letters"}
		}
	}

	if strings.TrimSpace(req.CustomerID) == "" {
		return &ValidationError{Field: "customer_id", Message: "is required"}
	}

	if strings.TrimSpace(req.IdempotencyKey) == "" {
		return &ValidationError{Field: "idempotency_key", Message: "is required"}
	}

	return nil
}

func newChargeID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("payflow: failed to generate charge ID: " + err.Error())
	}
	return "ch_" + hex.EncodeToString(b)
}
