package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStorePending(t *testing.T) {
	dir := t.TempDir()
	url := "https://example.com/webhook"
	payload := &BatchPayload{
		Event:          "batch.completed",
		ProcessedCount: 2,
		FailedCount:    0,
		Images:         []ImageEntry{{Source: "a.jpg", Destination: "a.webp"}, {Source: "b.png", Destination: "b.webp"}},
	}
	err := StorePending(dir, url, payload)
	if err != nil {
		t.Fatalf("StorePending: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	path := filepath.Join(dir, entries[0].Name())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var rec PendingRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rec.URL != url {
		t.Errorf("URL: got %q", rec.URL)
	}
	if rec.RetryCount != 0 {
		t.Errorf("RetryCount: got %d", rec.RetryCount)
	}
	if rec.Payload.ProcessedCount != 2 || len(rec.Payload.Images) != 2 {
		t.Errorf("Payload: ProcessedCount=%d Images=%d", rec.Payload.ProcessedCount, len(rec.Payload.Images))
	}
	if rec.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero")
	}
}

func TestRunRetryWorker_SuccessDeletesFile(t *testing.T) {
	dir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	payload := BatchPayload{Event: "batch.completed", ProcessedCount: 1, Images: []ImageEntry{}}
	rec := PendingRecord{URL: srv.URL, Payload: payload, RetryCount: 0, CreatedAt: time.Now()}
	recBytes, _ := json.MarshalIndent(rec, "", "  ")
	path := filepath.Join(dir, "20060102-120000-abc12345.json")
	if err := os.WriteFile(path, recBytes, 0644); err != nil {
		t.Fatalf("write pending file: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() {
		RunRetryWorker(ctx, RetryWorkerOptions{
			PendingDir:  dir,
			Interval:    10 * time.Millisecond,
			MaxRetries:  3,
			SendTimeout: time.Second,
		})
		close(done)
	}()
	time.Sleep(80 * time.Millisecond)
	cancel()
	<-done
	if _, err := os.Stat(path); err == nil {
		t.Error("expected pending file to be deleted after success")
	}
}
