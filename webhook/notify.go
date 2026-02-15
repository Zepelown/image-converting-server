package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// ImageEntry represents one converted image path (source and destination).
type ImageEntry struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// BatchPayload is the JSON body sent to the webhook URL on batch completion.
type BatchPayload struct {
	Event          string       `json:"event"`
	ProcessedCount int          `json:"processed_count"`
	FailedCount    int          `json:"failed_count"`
	Images         []ImageEntry `json:"images"`
}

// SendBulk POSTs the payload to the given URL as JSON. If url is empty, it returns nil without sending.
// Timeout is applied to the HTTP client. On failure, logs and returns the error.
func SendBulk(ctx context.Context, url string, payload *BatchPayload, timeout time.Duration) error {
	if url == "" {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[ERROR] Webhook: failed to marshal payload: %v", err)
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("[ERROR] Webhook: failed to create request: %v", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Webhook: POST failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("webhook returned status %d", resp.StatusCode)
		log.Printf("[ERROR] Webhook: %v for %s", err, url)
		return err
	}
	return nil
}
