package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	req.Header.Set("Accept", "*/*")

	// Preserve POST method and body on redirect (301/302 normally change to GET in stdlib).
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(redirReq *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			// Keep POST and body on redirect (default 301/302 would switch to GET).
			redirReq.Method = http.MethodPost
			redirReq.Body = io.NopCloser(bytes.NewReader(body))
			redirReq.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
			redirReq.ContentLength = int64(len(body))
			redirReq.Header.Set("Content-Type", "application/json")
			redirReq.Header.Set("Accept", "*/*")
			log.Printf("[INFO] Webhook: following redirect, re-sending POST to %s", redirReq.URL.String())
			return nil
		},
		Transport: &logTransport{next: http.DefaultTransport},
	}
	log.Printf("[INFO] Webhook: sending POST to %s", req.URL.String())
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Webhook: POST failed: %v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("webhook returned status %d", resp.StatusCode)
		log.Printf("[ERROR] Webhook: %v for %s (response Location: %v)", err, url, resp.Header.Get("Location"))
		return err
	}
	return nil
}

// logTransport logs the actual request method and URL before each round trip.
type logTransport struct{ next http.RoundTripper }

func (t *logTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.next == nil {
		t.next = http.DefaultTransport
	}
	log.Printf("[INFO] Webhook: outbound request %s %s", req.Method, req.URL.String())
	return t.next.RoundTrip(req)
}
