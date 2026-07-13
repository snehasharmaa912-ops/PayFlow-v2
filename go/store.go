package payflow

import (
	"fmt"
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
	eventLog         *EventLog
	riskEvaluator    RiskEvaluator
}

func NewStore() *Store {
	return &Store{
		charges:          make(map[string]*Charge),
		byIdempotencyKey: make(map[string]*Charge),
		ledger:           NewLedger(),
	}
}

func NewStoreWithLog(eventLog *EventLog) *Store {
	s := NewStore()
	s.eventLog = eventLog
	return s
}

func (s *Store) SetEventLog(eventLog *EventLog) {
	s.eventLog = eventLog
}

func (s *Store) SetRiskEvaluator(evaluator RiskEvaluator) {
	s.riskEvaluator = evaluator
}

func (s *Store) Ledger() *Ledger {
	return s.ledger
}

func (s *Store) CreateCharge(req CreateChargeRequest) (charge *Charge, created bool, err error) {
	if verr := validateCreateChargeRequest(req); verr != nil {
		return nil, false, verr
	}

	charge = &Charge{
		ID:             newChargeID(),
		Amount:         req.Amount,
		Currency:       strings.ToUpper(strings.TrimSpace(req.Currency)),
		CustomerID:     strings.TrimSpace(req.CustomerID),
		IdempotencyKey: strings.TrimSpace(req.IdempotencyKey),
		Status:         "succeeded",
		CreatedAt:      time.Now().UTC(),
	}

	if charge.Amount > LargeChargeThreshold && s.riskEvaluator != nil {
		decision, reason, evalErr := s.riskEvaluator.Evaluate(charge)
		if evalErr != nil {
			decision = "review"
			reason = "risk engine unavailable: " + evalErr.Error()
		}
		charge.RiskDecision = decision
		charge.RiskReason = reason
		charge.Status = statusForDecision(decision)
	}

	s.mu.Lock()
	if existing, ok := s.byIdempotencyKey[req.IdempotencyKey]; ok {
		s.mu.Unlock()
		return existing, false, nil
	}

	if s.eventLog != nil {
		if err := s.eventLog.Append(Event{Type: EventChargeCreated, Charge: charge}); err != nil {
			s.mu.Unlock()
			return nil, false, fmt.Errorf("writing to event log: %w", err)
		}
	}

	s.charges[charge.ID] = charge
	s.byIdempotencyKey[charge.IdempotencyKey] = charge
	s.mu.Unlock()

	if charge.Status != "declined" {
		s.ledger.RecordCharge(charge)
	}

	return charge, true, nil
}

func (s *Store) applyChargeCreated(charge *Charge) {
	s.mu.Lock()
	s.charges[charge.ID] = charge
	s.byIdempotencyKey[charge.IdempotencyKey] = charge
	s.mu.Unlock()

	if charge.Status != "declined" {
		s.ledger.RecordCharge(charge)
	}
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
