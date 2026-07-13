package payflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestConcurrentDuplicateRequests_ExactlyOneChargeCreated(t *testing.T) {
	store := NewStore()
	api := NewAPI(store)
	server := httptest.NewServer(api.Routes())
	defer server.Close()

	const concurrentRequests = 50
	req := CreateChargeRequest{
		Amount:         5000,
		Currency:       "USD",
		CustomerID:     "cus_race",
		IdempotencyKey: "race-key",
	}
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshaling request: %v", err)
	}

	var wg sync.WaitGroup
	results := make([]int, concurrentRequests)
	chargeIDs := make([]string, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			resp, err := http.Post(server.URL+"/charges", "application/json", bytes.NewReader(body))
			if err != nil {
				t.Errorf("request %d failed: %v", idx, err)
				return
			}
			defer resp.Body.Close()

			var charge Charge
			if err := json.NewDecoder(resp.Body).Decode(&charge); err != nil {
				t.Errorf("request %d: decoding response: %v", idx, err)
				return
			}

			results[idx] = resp.StatusCode
			chargeIDs[idx] = charge.ID
		}(i)
	}
	wg.Wait()

	listResp, err := http.Get(server.URL + "/charges")
	if err != nil {
		t.Fatalf("listing charges: %v", err)
	}
	defer listResp.Body.Close()

	var charges []Charge
	if err := json.NewDecoder(listResp.Body).Decode(&charges); err != nil {
		t.Fatalf("decoding charge list: %v", err)
	}

	if len(charges) != 1 {
		t.Fatalf("expected exactly 1 charge after %d concurrent identical requests, got %d", concurrentRequests, len(charges))
	}

	firstID := chargeIDs[0]
	for i, id := range chargeIDs {
		if id != firstID {
			t.Errorf("request %d returned charge ID %q, expected all requests to return %q", i, id, firstID)
		}
	}

	createdCount := 0
	for _, status := range results {
		if status == http.StatusCreated {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Errorf("expected exactly 1 request to get 201 Created, got %d", createdCount)
	}

	ok, sum := store.Ledger().Verify()
	if !ok {
		t.Fatalf("expected ledger to be balanced after concurrent requests, sum was %d", sum)
	}
	if len(store.Ledger().Entries()) != 2 {
		t.Fatalf("expected exactly 2 ledger entries (1 debit + 1 credit), got %d", len(store.Ledger().Entries()))
	}
}
