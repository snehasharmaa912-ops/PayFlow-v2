package payflow

import (
	"sort"
	"strings"
	"sync"
	"time"
)

type Store struct {
	mu               sync.RWMutex
	charges          map[string]*Charge
	byIdempotencyKey map[string]*Charge
	ledger           *Ledger
}

func NewStore() *Store {
	return &Store{
		charges:          make(map[string]*Charge),
		byIdempotencyKey: make(map[string]*Charge),
		ledger:           NewLedger(),
	}
}

func (s *Store) Ledger() *Ledger {
	return s.ledger
}

func (s *Store) CreateCharge(req CreateChargeRequest) (charge *Charge, created bool, err error) {
	if verr := validateCreateChargeRequest(req); verr != nil {
		return nil, false, verr
	}

	s.mu.RLock()
	if existing, ok := s.byIdempotencyKey[req.IdempotencyKey]; ok {
		s.mu.RUnlock()
		return existing, false, nil
	}
	s.mu.RUnlock()

	charge = &Charge{
		ID:             newChargeID(),
		Amount:         req.Amount,
		Currency:       strings.ToUpper(strings.TrimSpace(req.Currency)),
		CustomerID:     strings.TrimSpace(req.CustomerID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		Status:         "succeeded",
		CreatedAt:      time.Now().UTC(),
	}

	s.mu.Lock()
	s.charges[charge.ID] = charge
	s.byIdempotencyKey[charge.IdempotencyKey] = charge
	s.mu.Unlock()
	s.ledger.RecordCharge(charge)

	return charge, true, nil
}

func (s *Store) GetCharge(id string) (*Charge, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.charges[id]
	return c, ok
}

func (s *Store) ListCharges() []*Charge {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Charge, 0, len(s.charges))
	for _, c := range s.charges {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.Before(out[j].CreatedAt)
	})
	return out
}
