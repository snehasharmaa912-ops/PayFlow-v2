package payflow

import (
	"encoding/json"
	"errors"
	"net/http"
)

type API struct {
	store *Store
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
	return mux
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
