package payflow

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type EventType string

const (
	EventChargeCreated EventType = "charge_created"
)

type Event struct {
	Type   EventType `json:"type"`
	Charge *Charge   `json:"charge"`
}

type EventLog struct {
	mu   sync.Mutex
	file *os.File
	w    *bufio.Writer
}

func OpenEventLog(path string) (*EventLog, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("opening event log: %w", err)
	}
	return &EventLog{
		file: f,
		w:    bufio.NewWriter(f),
	}, nil
}

func (l *EventLog) Append(event Event) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshaling event: %w", err)
	}

	if _, err := l.w.Write(data); err != nil {
		return fmt.Errorf("writing event: %w", err)
	}
	if err := l.w.WriteByte('\n'); err != nil {
		return fmt.Errorf("writing newline: %w", err)
	}
	if err := l.w.Flush(); err != nil {
		return fmt.Errorf("flushing event log: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("syncing event log: %w", err)
	}

	return nil
}

func (l *EventLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.w.Flush(); err != nil {
		return err
	}
	return l.file.Close()
}

func ReadEvents(path string) ([]Event, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("opening event log: %w", err)
	}
	defer f.Close()

	var events []Event
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var event Event
		if err := json.Unmarshal(line, &event); err != nil {
			return nil, fmt.Errorf("decoding event: %w", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading event log: %w", err)
	}
	return events, nil
}

func ReplayFromLog(path string) (*Store, error) {
	events, err := ReadEvents(path)
	if err != nil {
		return nil, err
	}

	store := NewStore()
	for _, event := range events {
		switch event.Type {
		case EventChargeCreated:
			if event.Charge == nil {
				return nil, fmt.Errorf("replay: charge_created event missing charge data")
			}
			store.applyChargeCreated(event.Charge)
		default:
			return nil, fmt.Errorf("replay: unknown event type %q", event.Type)
		}
	}

	return store, nil
}
