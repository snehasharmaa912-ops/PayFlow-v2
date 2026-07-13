package payflow

import (
	"errors"
	"testing"
)

type fakeRiskEvaluator struct {
	decision string
	reason   string
	err      error
}

func (f *fakeRiskEvaluator) Evaluate(charge *Charge) (string, string, error) {
	return f.decision, f.reason, f.err
}

func TestCreateCharge_SkipsRiskCheckBelowThreshold(t *testing.T) {
	store := NewStore()
	store.SetRiskEvaluator(&fakeRiskEvaluator{decision: "decline", reason: "should not be called"})

	charge, _, err := store.CreateCharge(CreateChargeRequest{
		Amount: 500, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charge.Status != "succeeded" {
		t.Errorf("expected small charge to skip risk check and succeed, got status %q", charge.Status)
	}
	if charge.RiskDecision != "" {
		t.Errorf("expected no risk decision for a small charge, got %q", charge.RiskDecision)
	}
}

func TestCreateCharge_ApprovedLargeCharge(t *testing.T) {
	store := NewStore()
	store.SetRiskEvaluator(&fakeRiskEvaluator{decision: "approve", reason: "within limits"})

	charge, _, err := store.CreateCharge(CreateChargeRequest{
		Amount: 200000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charge.Status != "succeeded" {
		t.Errorf("expected approved charge to succeed, got status %q", charge.Status)
	}
}

func TestCreateCharge_ReviewedLargeCharge(t *testing.T) {
	store := NewStore()
	store.SetRiskEvaluator(&fakeRiskEvaluator{decision: "review", reason: "over review threshold"})

	charge, _, err := store.CreateCharge(CreateChargeRequest{
		Amount: 200000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charge.Status != "pending_review" {
		t.Errorf("expected status pending_review, got %q", charge.Status)
	}

	ok, sum := store.Ledger().Verify()
	if !ok {
		t.Fatalf("expected ledger balanced, sum was %d", sum)
	}
	if len(store.Ledger().Entries()) != 2 {
		t.Errorf("expected pending_review charge to still post to the ledger, got %d entries", len(store.Ledger().Entries()))
	}
}

func TestCreateCharge_DeclinedLargeChargeDoesNotPostToLedger(t *testing.T) {
	store := NewStore()
	store.SetRiskEvaluator(&fakeRiskEvaluator{decision: "decline", reason: "amount too high"})

	charge, _, err := store.CreateCharge(CreateChargeRequest{
		Amount: 5000000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charge.Status != "declined" {
		t.Errorf("expected status declined, got %q", charge.Status)
	}
	if len(store.Ledger().Entries()) != 0 {
		t.Errorf("expected a declined charge to NOT post to the ledger, got %d entries", len(store.Ledger().Entries()))
	}

	ok, _ := store.Ledger().Verify()
	if !ok {
		t.Fatal("expected empty ledger to still be trivially balanced")
	}
}

func TestCreateCharge_RiskEngineFailureFailsSafeToReview(t *testing.T) {
	store := NewStore()
	store.SetRiskEvaluator(&fakeRiskEvaluator{err: errors.New("connection refused")})

	charge, _, err := store.CreateCharge(CreateChargeRequest{
		Amount: 200000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if charge.Status != "pending_review" {
		t.Errorf("expected risk engine failure to fail safe to pending_review, got %q", charge.Status)
	}
}
