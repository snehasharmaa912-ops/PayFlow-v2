package payflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestAPI() *API {
	return NewAPI(NewStore())
}

func doRequest(t *testing.T, handler http.Handler, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encoding request body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestCreateCharge_Success(t *testing.T) {
	handler := newTestAPI().Routes()

	req := CreateChargeRequest{
		Amount:         5000,
		Currency:       "usd",
		CustomerID:     "cus_123",
		IdempotencyKey: "key-1",
	}
	rec := doRequest(t, handler, http.MethodPost, "/charges", req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", rec.Code, rec.Body.String())
	}

	var charge Charge
	if err := json.Unmarshal(rec.Body.Bytes(), &charge); err != nil {
		t.Fatalf("decoding response: %v", err)
	}
	if charge.Amount != 5000 {
		t.Errorf("expected amount 5000, got %d", charge.Amount)
	}
	if charge.Currency != "USD" {
		t.Errorf("expected currency normalized to USD, got %q", charge.Currency)
	}
	if charge.Status != "succeeded" {
		t.Errorf("expected status succeeded, got %q", charge.Status)
	}
	if charge.ID == "" {
		t.Error("expected a non-empty charge ID")
	}
}

func TestCreateCharge_Validation(t *testing.T) {
	handler := newTestAPI().Routes()

	cases := []struct {
		name string
		req  CreateChargeRequest
	}{
		{"zero amount", CreateChargeRequest{Amount: 0, Currency: "USD", CustomerID: "c1", IdempotencyKey: "k1"}},
		{"negative amount", CreateChargeRequest{Amount: -100, Currency: "USD", CustomerID: "c1", IdempotencyKey: "k1"}},
		{"bad currency length", CreateChargeRequest{Amount: 100, Currency: "US", CustomerID: "c1", IdempotencyKey: "k1"}},
		{"non-letter currency", CreateChargeRequest{Amount: 100, Currency: "U$D", CustomerID: "c1", IdempotencyKey: "k1"}},
		{"missing customer id", CreateChargeRequest{Amount: 100, Currency: "USD", CustomerID: "", IdempotencyKey: "k1"}},
		{"missing idempotency key", CreateChargeRequest{Amount: 100, Currency: "USD", CustomerID: "c1", IdempotencyKey: ""}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := doRequest(t, handler, http.MethodPost, "/charges", tc.req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCreateCharge_IdempotentReplay(t *testing.T) {
	handler := newTestAPI().Routes()

	req := CreateChargeRequest{
		Amount:         2500,
		Currency:       "EUR",
		CustomerID:     "cus_42",
		IdempotencyKey: "same-key",
	}

	first := doRequest(t, handler, http.MethodPost, "/charges", req)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected first request to return 201, got %d", first.Code)
	}
	var firstCharge Charge
	if err := json.Unmarshal(first.Body.Bytes(), &firstCharge); err != nil {
		t.Fatalf("decoding first response: %v", err)
	}

	second := doRequest(t, handler, http.MethodPost, "/charges", req)
	if second.Code != http.StatusOK {
		t.Fatalf("expected replay to return 200, got %d", second.Code)
	}
	var secondCharge Charge
	if err := json.Unmarshal(second.Body.Bytes(), &secondCharge); err != nil {
		t.Fatalf("decoding second response: %v", err)
	}

	if firstCharge.ID != secondCharge.ID {
		t.Fatalf("expected replay to return the same charge ID, got %q and %q", firstCharge.ID, secondCharge.ID)
	}

	list := doRequest(t, handler, http.MethodGet, "/charges", nil)
	var charges []Charge
	if err := json.Unmarshal(list.Body.Bytes(), &charges); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	if len(charges) != 1 {
		t.Fatalf("expected exactly 1 charge after a duplicate request, got %d", len(charges))
	}
}

func TestGetCharge(t *testing.T) {
	handler := newTestAPI().Routes()

	created := doRequest(t, handler, http.MethodPost, "/charges", CreateChargeRequest{
		Amount: 100, Currency: "USD", CustomerID: "c1", IdempotencyKey: "k1",
	})
	var charge Charge
	if err := json.Unmarshal(created.Body.Bytes(), &charge); err != nil {
		t.Fatalf("decoding create response: %v", err)
	}

	rec := doRequest(t, handler, http.MethodGet, "/charges/"+charge.ID, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	missing := doRequest(t, handler, http.MethodGet, "/charges/does-not-exist", nil)
	if missing.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing charge, got %d", missing.Code)
	}
}

func TestListCharges(t *testing.T) {
	handler := newTestAPI().Routes()

	doRequest(t, handler, http.MethodPost, "/charges", CreateChargeRequest{
		Amount: 100, Currency: "USD", CustomerID: "c1", IdempotencyKey: "k1",
	})
	doRequest(t, handler, http.MethodPost, "/charges", CreateChargeRequest{
		Amount: 200, Currency: "USD", CustomerID: "c2", IdempotencyKey: "k2",
	})

	rec := doRequest(t, handler, http.MethodGet, "/charges", nil)
	var charges []Charge
	if err := json.Unmarshal(rec.Body.Bytes(), &charges); err != nil {
		t.Fatalf("decoding list response: %v", err)
	}
	if len(charges) != 2 {
		t.Fatalf("expected 2 charges, got %d", len(charges))
	}
}
