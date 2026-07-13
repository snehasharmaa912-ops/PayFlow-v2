package payflow

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type Entry struct {
	ID        string    `json:"id"`
	ChargeID  string    `json:"charge_id"`
	Account   string    `json:"account"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type Ledger struct {
	mu       sync.RWMutex
	entries  []Entry
	balances map[string]int64
}

func NewLedger() *Ledger {
	return &Ledger{
		balances: make(map[string]int64),
	}
}

func (l *Ledger) RecordCharge(charge *Charge) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	customerAccount := "customer:" + charge.CustomerID
	platformAccount := "platform:revenue"

	debit := Entry{
		ID:        newEntryID(),
		ChargeID:  charge.ID,
		Account:   customerAccount,
		Amount:    -charge.Amount,
		CreatedAt: now,
	}
	credit := Entry{
		ID:        newEntryID(),
		ChargeID:  charge.ID,
		Account:   platformAccount,
		Amount:    charge.Amount,
		CreatedAt: now,
	}

	l.entries = append(l.entries, debit, credit)
	l.balances[customerAccount] += debit.Amount
	l.balances[platformAccount] += credit.Amount
}

func (l *Ledger) Balance(account string) int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.balances[account]
}

func (l *Ledger) Balances() map[string]int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make(map[string]int64, len(l.balances))
	for k, v := range l.balances {
		out[k] = v
	}
	return out
}

func (l *Ledger) Entries() []Entry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Entry, len(l.entries))
	copy(out, l.entries)
	return out
}

func (l *Ledger) Verify() (balanced bool, sum int64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var total int64
	for _, e := range l.entries {
		total += e.Amount
	}
	return total == 0, total
}

func newEntryID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic("payflow: failed to generate entry ID: " + err.Error())
	}
	return "le_" + hex.EncodeToString(b)
}
