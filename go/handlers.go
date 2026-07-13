package payflow

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
)

type API struct {
	store *Store
	logPath string
}

func (a *API) WithLogPath(path string) *API {
	a.logPath = path
	return a
}

func NewAPI(store *Store) *API {
	return &API{store: store}
}

func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /charges", a.handleCreateCharge)
	mux.HandleFunc("GET /charges", a.handleListCharges)
	mux.HandleFunc("GET /charges/{id}", a.handleGetCharge)
	mux.HandleFunc("GET /healthz", a.handleHealth)
	mux.HandleFunc("GET /ledger", a.handleLedger)
	mux.HandleFunc("GET /debug/verify-replay", a.handleVerifyReplay)
	return mux
}

type ledgerResponse struct {
	Balances map[string]int64 `json:"balances"`
	Balanced bool              `json:"balanced"`
	Sum      int64             `json:"sum"`
}

func (a *API) handleLedger(w http.ResponseWriter, r *http.Request) {
	ledger := a.store.Ledger()
	balanced, sum := ledger.Verify()
	writeJSON(w, http.StatusOK, ledgerResponse{
		Balances: ledger.Balances(),
		Balanced: balanced,
		Sum:      sum,
	})
}

func (a *API) handleCreateCharge(w http.ResponseWriter, r *http.Request) {
	var req CreateChargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body must be valid JSON")
		return
	}

	charge, created, err := a.store.CreateCharge(req)
	if err != nil {
		var verr *ValidationError
		if errors.As(err, &verr) {
			writeError(w, http.StatusBadRequest, verr.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, charge)
}

func (a *API) handleVerifyReplay(w http.ResponseWriter, r *http.Request) {
	if a.logPath == "" {
		writeError(w, http.StatusNotImplemented, "no event log configured for this server")
		return
	}

	replayed, err := ReplayFromLog(a.logPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "replay failed: "+err.Error())
		return
	}

	liveOK, liveSum := a.store.Ledger().Verify()
	replayOK, replaySum := replayed.Ledger().Verify()
	matches := reflect.DeepEqual(a.store.Ledger().Balances(), replayed.Ledger().Balances()) &&
		len(a.store.ListCharges()) == len(replayed.ListCharges())

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"matches":             matches,
		"live_balanced":       liveOK,
		"live_sum":            liveSum,
		"replay_balanced":     replayOK,
		"replay_sum":          replaySum,
		"live_charge_count":   len(a.store.ListCharges()),
		"replay_charge_count": len(replayed.ListCharges()),
	})
}

func (a *API) handleGetCharge(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	charge, ok := a.store.GetCharge(id)
	if !ok {
		writeError(w, http.StatusNotFound, "charge not found")
		return
	}
	writeJSON(w, http.StatusOK, charge)
}

func (a *API) handleListCharges(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.store.ListCharges())
}

func (a *API) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
