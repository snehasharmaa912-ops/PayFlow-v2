package payflow

import "testing"

func TestLedger_RecordChargeCreatesBalancedEntries(t *testing.T) {
	ledger := NewLedger()
	charge := &Charge{ID: "ch_1", Amount: 5000, CustomerID: "cus_1"}

	ledger.RecordCharge(charge)

	entries := ledger.Entries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	ok, sum := ledger.Verify()
	if !ok {
		t.Fatalf("expected ledger to be balanced, sum was %d", sum)
	}
}

func TestLedger_BalanceAfterMultipleCharges(t *testing.T) {
	ledger := NewLedger()

	charges := []*Charge{
		{ID: "ch_1", Amount: 5000, CustomerID: "cus_1"},
		{ID: "ch_2", Amount: 2500, CustomerID: "cus_1"},
		{ID: "ch_3", Amount: 1000, CustomerID: "cus_2"},
	}

	for _, c := range charges {
		ledger.RecordCharge(c)
	}

	cus1Balance := ledger.Balance("customer:cus_1")
	if cus1Balance != -7500 {
		t.Errorf("expected cus_1 balance -7500, got %d", cus1Balance)
	}

	cus2Balance := ledger.Balance("customer:cus_2")
	if cus2Balance != -1000 {
		t.Errorf("expected cus_2 balance -1000, got %d", cus2Balance)
	}

	platformBalance := ledger.Balance("platform:revenue")
	if platformBalance != 8500 {
		t.Errorf("expected platform balance 8500, got %d", platformBalance)
	}

	ok, sum := ledger.Verify()
	if !ok {
		t.Fatalf("expected ledger to be balanced after 3 charges, sum was %d", sum)
	}
}

func TestLedger_VerifyCatchesImbalance(t *testing.T) {
	ledger := NewLedger()
	ledger.RecordCharge(&Charge{ID: "ch_1", Amount: 5000, CustomerID: "cus_1"})

	ledger.mu.Lock()
	ledger.entries[1].Amount = 4000
	ledger.mu.Unlock()

	ok, sum := ledger.Verify()
	if ok {
		t.Fatal("expected Verify to catch a tampered, unbalanced ledger")
	}
	if sum == 0 {
		t.Error("expected a non-zero sum for an unbalanced ledger")
	}
}
