package payflow

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"
)

type WebhookEvent struct {
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Charge    *Charge   `json:"charge"`
}

type WebhookDispatcher struct {
	URL        string
	Secret     string
	Client     *http.Client
	MaxRetries int
}

func NewWebhookDispatcher(url, secret string) *WebhookDispatcher {
	return &WebhookDispatcher{
		URL:        url,
		Secret:     secret,
		Client:     &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 3,
	}
}

func (d *WebhookDispatcher) sign(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(d.Secret))
	mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

func (d *WebhookDispatcher) Dispatch(eventType string, charge *Charge) {
	if d.URL == "" {
		return
	}

	event := WebhookEvent{
		Type:      eventType,
		CreatedAt: time.Now().UTC(),
		Charge:    charge,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	go d.deliverWithRetry(payload)
}

func (d *WebhookDispatcher) deliverWithRetry(payload []byte) {
	backoff := 500 * time.Millisecond

	for attempt := 0; attempt <= d.MaxRetries; attempt++ {
		req, err := http.NewRequest(http.MethodPost, d.URL, bytes.NewReader(payload))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Payflow-Signature", d.sign(payload))

		resp, err := d.Client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
		}

		time.Sleep(backoff)
		backoff *= 2
	}
}
