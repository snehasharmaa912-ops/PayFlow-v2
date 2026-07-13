package payflow

import "testing"

func TestStore_CreateChargeUpdatesLedger(t *testing.T) {
	store := NewStore()

	reqs := []CreateChargeRequest{
		{Amount: 5000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1"},
		{Amount: 2500, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k2"},
		{Amount: 1000, Currency: "USD", CustomerID: "cus_2", IdempotencyKey: "k3"},
	}

	for _, req := range reqs {
		if _, _, err := store.CreateCharge(req); err != nil {
			t.Fatalf("unexpected error creating charge: %v", err)
		}
	}

	ok, sum := store.Ledger().Verify()
	if !ok {
		t.Fatalf("expected ledger to be balanced, sum was %d", sum)
	}

	if store.Ledger().Balance("customer:cus_1") != -7500 {
		t.Errorf("expected cus_1 balance -7500, got %d", store.Ledger().Balance("customer:cus_1"))
	}
	if store.Ledger().Balance("platform:revenue") != 8500 {
		t.Errorf("expected platform balance 8500, got %d", store.Ledger().Balance("platform:revenue"))
	}
}

func TestStore_IdempotentReplayDoesNotDoublePostToLedger(t *testing.T) {
	store := NewStore()
	req := CreateChargeRequest{Amount: 5000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "same-key"}

	if _, _, err := store.CreateCharge(req); err != nil {
		t.Fatalf("unexpected error on first create: %v", err)
	}
	if _, _, err := store.CreateCharge(req); err != nil {
		t.Fatalf("unexpected error on replay: %v", err)
	}

	if len(store.Ledger().Entries()) != 2 {
		t.Fatalf("expected exactly 2 ledger entries after a duplicate request, got %d", len(store.Ledger().Entries()))
	}
}
