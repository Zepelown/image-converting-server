package webhook

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const defaultPendingDir = "data/webhook_pending"

// PendingRecord is the JSON structure stored per failed webhook for retry.
type PendingRecord struct {
	URL        string       `json:"url"`
	Payload    BatchPayload `json:"payload"`
	RetryCount int          `json:"retry_count"`
	CreatedAt  time.Time    `json:"created_at"`
}

// StorePending writes a failed webhook payload to a file in pendingDir for later retry.
// If pendingDir is empty, defaultPendingDir is used. Creates the directory if needed.
// Uses a temp file + rename for atomic write.
func StorePending(pendingDir, url string, payload *BatchPayload) error {
	if pendingDir == "" {
		pendingDir = defaultPendingDir
	}
	if err := os.MkdirAll(pendingDir, 0755); err != nil {
		return err
	}
	shortID, err := randomShortID()
	if err != nil {
		return err
	}
	name := fmt.Sprintf("%s-%s.json", time.Now().Format("20060102-150405"), shortID)
	path := filepath.Join(pendingDir, name)
	rec := PendingRecord{
		URL:        url,
		Payload:    *payload,
		RetryCount: 0,
		CreatedAt:  time.Now(),
	}
	data, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

func randomShortID() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// RetryWorkerOptions configures the retry worker.
type RetryWorkerOptions struct {
	PendingDir     string
	Interval       time.Duration
	MaxRetries     int
	SendTimeout    time.Duration
	DeadLetterDir  string // optional; if set, move here on max retries instead of deleting
}

// RunRetryWorker runs a loop that periodically scans pendingDir for JSON files,
// sends each via SendBulk, and removes or updates the file on success or failure.
// It exits when ctx is cancelled.
func RunRetryWorker(ctx context.Context, opts RetryWorkerOptions) {
	if opts.PendingDir == "" {
		opts.PendingDir = defaultPendingDir
	}
	if opts.Interval == 0 {
		opts.Interval = 5 * time.Minute
	}
	if opts.SendTimeout == 0 {
		opts.SendTimeout = 10 * time.Second
	}
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()
	// Run first pass soon so newly stored pendings get an immediate retry
	runRetryPass(ctx, opts)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runRetryPass(ctx, opts)
		}
	}
}

func runRetryPass(ctx context.Context, opts RetryWorkerOptions) {
	entries, err := os.ReadDir(opts.PendingDir)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		log.Printf("[WARN] Webhook retry: read dir %s: %v", opts.PendingDir, err)
		return
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(name), ".json") || strings.HasSuffix(strings.ToLower(name), ".tmp") {
			continue
		}
		path := filepath.Join(opts.PendingDir, name)
		rec, err := readPendingFile(path)
		if err != nil {
			log.Printf("[WARN] Webhook retry: read %s: %v", path, err)
			continue
		}
		err = SendBulk(ctx, rec.URL, &rec.Payload, opts.SendTimeout)
		if err == nil {
			if removeErr := os.Remove(path); removeErr != nil {
				log.Printf("[WARN] Webhook retry: remove %s: %v", path, removeErr)
			}
			continue
		}
		rec.RetryCount++
		if rec.RetryCount >= opts.MaxRetries {
			if opts.DeadLetterDir != "" {
				_ = os.MkdirAll(opts.DeadLetterDir, 0755)
				dest := filepath.Join(opts.DeadLetterDir, filepath.Base(path))
				if moveErr := os.Rename(path, dest); moveErr != nil {
					log.Printf("[WARN] Webhook retry: move to dead letter %s: %v", dest, moveErr)
					_ = os.Remove(path)
				} else {
					log.Printf("[INFO] Webhook retry: moved to dead letter after %d retries: %s", opts.MaxRetries, dest)
				}
			} else {
				if removeErr := os.Remove(path); removeErr != nil {
					log.Printf("[WARN] Webhook retry: remove %s: %v", path, removeErr)
				} else {
					log.Printf("[INFO] Webhook retry: removed after %d retries: %s", opts.MaxRetries, path)
				}
			}
			continue
		}
		data, marshalErr := json.MarshalIndent(rec, "", "  ")
		if marshalErr != nil {
			log.Printf("[WARN] Webhook retry: marshal %s: %v", path, marshalErr)
			continue
		}
		tmpPath := path + ".tmp"
		if writeErr := os.WriteFile(tmpPath, data, 0644); writeErr != nil {
			log.Printf("[WARN] Webhook retry: write %s: %v", tmpPath, writeErr)
			continue
		}
		if renameErr := os.Rename(tmpPath, path); renameErr != nil {
			log.Printf("[WARN] Webhook retry: rename %s: %v", path, renameErr)
			_ = os.Remove(tmpPath)
		}
	}
}

func readPendingFile(path string) (*PendingRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec PendingRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, err
	}
	return &rec, nil
}
