package internal

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

type webhookPayload struct {
	ID        string `json:"id"`        // es. "595068606-us"
	Timestamp string `json:"timestamp"` // RFC3339 UTC
	ErrorType string `json:"errorType"` // es. "network_timeout", "http_status_500", "decode_error"
}

func NotifyWebhook(url, id, errType string) error {
	if url == "" {
		return nil // webhook disabled
	}
	body := webhookPayload{
		ID:        id,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		ErrorType: errType,
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// non falliamo il chiamante se 4xx/5xx del webhook: è best-effort
	return nil
}
