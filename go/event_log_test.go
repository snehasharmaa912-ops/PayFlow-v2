package payflow

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestEventLog_AppendAndReadEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.log")

	log, err := OpenEventLog(path)
	if err != nil {
		t.Fatalf("opening event log: %v", err)
	}
	defer log.Close()

	charge := &Charge{ID: "ch_1", Amount: 100, CustomerID: "cus_1", IdempotencyKey: "k1"}
	if err := log.Append(Event{Type: EventChargeCreated, Charge: charge}); err != nil {
		t.Fatalf("appending event: %v", err)
	}

	events, err := ReadEvents(path)
	if err != nil {
		t.Fatalf("reading events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Charge.ID != "ch_1" {
		t.Errorf("expected charge ID ch_1, got %q", events[0].Charge.ID)
	}
}

func TestReplayFromLog_MatchesLiveStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.log")

	eventLog, err := OpenEventLog(path)
	if err != nil {
		t.Fatalf("opening event log: %v", err)
	}

	live := NewStoreWithLog(eventLog)

	reqs := []CreateChargeRequest{
		{Amount: 5000, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k1"},
		{Amount: 2500, Currency: "USD", CustomerID: "cus_1", IdempotencyKey: "k2"},
		{Amount: 1000, Currency: "USD", CustomerID: "cus_2", IdempotencyKey: "k3"},
	}
	for _, req := range reqs {
		if _, _, err := live.CreateCharge(req); err != nil {
			t.Fatalf("creating charge: %v", err)
		}
	}
	if _, _, err := live.CreateCharge(reqs[0]); err != nil {
		t.Fatalf("replaying idempotent request: %v", err)
	}
	eventLog.Close()

	replayed, err := ReplayFromLog(path)
	if err != nil {
		t.Fatalf("replaying from log: %v", err)
	}

	liveCharges := live.ListCharges()
	replayedCharges := replayed.ListCharges()
	if len(liveCharges) != len(replayedCharges) {
		t.Fatalf("expected %d charges after replay, got %d", len(liveCharges), len(replayedCharges))
	}
	sort.Slice(liveCharges, func(i, j int) bool { return liveCharges[i].ID < liveCharges[j].ID })
	sort.Slice(replayedCharges, func(i, j int) bool { return replayedCharges[i].ID < replayedCharges[j].ID })
	for i := range liveCharges {
		if liveCharges[i].ID != replayedCharges[i].ID || liveCharges[i].Amount != replayedCharges[i].Amount {
			t.Errorf("charge mismatch at index %d: live=%+v replayed=%+v", i, liveCharges[i], replayedCharges[i])
		}
	}

	liveBalances := live.Ledger().Balances()
	replayedBalances := replayed.Ledger().Balances()
	if !reflect.DeepEqual(liveBalances, replayedBalances) {
		t.Errorf("balances mismatch after replay:\nlive:     %+v\nreplayed: %+v", liveBalances, replayedBalances)
	}

	liveOK, liveSum := live.Ledger().Verify()
	replayedOK, replayedSum := replayed.Ledger().Verify()
	if !liveOK || !replayedOK {
		t.Fatalf("expected both ledgers balanced, live=%v(%d) replayed=%v(%d)", liveOK, liveSum, replayedOK, replayedSum)
	}
}

func TestReplayFromLog_EmptyLogProducesEmptyStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.log")

	store, err := ReplayFromLog(path)
	if err != nil {
		t.Fatalf("replaying from nonexistent log: %v", err)
	}
	if len(store.ListCharges()) != 0 {
		t.Errorf("expected 0 charges from empty log, got %d", len(store.ListCharges()))
	}
}
