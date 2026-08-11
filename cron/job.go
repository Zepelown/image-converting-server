package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"image-converting-server/config"
	"image-converting-server/processor"
	"image-converting-server/r2"
	"image-converting-server/state"
	"image-converting-server/webhook"

	"github.com/robfig/cron/v3"
)

// Job represents the cron job scheduler
type Job struct {
	cron      *cron.Cron
	cfg       *config.Config
	r2Client  r2.StorageClient
	r2Clients map[string]r2.StorageClient
	processor *processor.Processor
	statePath string
	lockPath  string
}

// NewJob creates a new Job instance
func NewJob(cfg *config.Config, r2Client r2.StorageClient, proc *processor.Processor, statePath string) *Job {
	return &Job{
		cron:      cron.New(),
		cfg:       cfg,
		r2Client:  r2Client,
		r2Clients: map[string]r2.StorageClient{cfg.R2.Bucket: r2Client},
		processor: proc,
		statePath: statePath,
		lockPath:  filepath.Join(filepath.Dir(statePath), ".lock"),
	}
}

// NewMultiBucketJob creates a cron job that processes each configured bucket.
func NewMultiBucketJob(cfg *config.Config, r2Clients map[string]r2.StorageClient, proc *processor.Processor, statePath string) *Job {
	return &Job{
		cron:      cron.New(),
		cfg:       cfg,
		r2Clients: r2Clients,
		processor: proc,
		statePath: statePath,
		lockPath:  filepath.Join(filepath.Dir(statePath), ".lock"),
	}
}

// Start registers and starts the cron job
func (j *Job) Start() error {
	if !j.cfg.Cron.Enabled {
		log.Println("[INFO] Cron job is disabled")
		return nil
	}

	_, err := j.cron.AddFunc(j.cfg.Cron.Schedule, func() {
		j.ProcessImages()
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	j.cron.Start()
	log.Printf("[INFO] Cron job started with schedule: %s", j.cfg.Cron.Schedule)
	return nil
}

// Stop stops the cron job
func (j *Job) Stop() {
	j.cron.Stop()
}

// ProcessImages runs the image conversion process
func (j *Job) ProcessImages() {
	// 1. Check/Create Lock
	if err := j.acquireLock(); err != nil {
		log.Printf("[ERROR] Failed to acquire lock: %v", err)
		return
	}
	defer j.releaseLock()

	log.Println("[INFO] Cron job execution started")
	startTime := time.Now()
	ctx := context.Background()

	processedCount := 0
	failedCount := 0
	var converted []webhook.ImageEntry

	for _, bucket := range j.bucketNames() {
		client, ok := j.r2Clients[bucket]
		if !ok {
			log.Printf("[ERROR] R2 client for bucket %s is not configured", bucket)
			failedCount++
			continue
		}

		bucketProcessed, bucketFailed, bucketConverted := j.processBucket(ctx, bucket, client, startTime)
		processedCount += bucketProcessed
		failedCount += bucketFailed
		converted = append(converted, bucketConverted...)
	}

	log.Printf("[INFO] Cron job execution completed. Processed: %d, Failed: %d, Duration: %v",
		processedCount, failedCount, time.Since(startTime))

	if j.cfg.Webhook.URL != "" {
		payload := &webhook.BatchPayload{
			Event:          "batch.completed",
			ProcessedCount: processedCount,
			FailedCount:    failedCount,
			Images:         converted,
		}
		if err := webhook.SendBulk(ctx, j.cfg.Webhook.URL, payload, 10*time.Second); err != nil {
			log.Printf("[WARN] Webhook send failed (batch completed successfully): %v", err)
			if j.cfg.Webhook.RetryEnabled {
				if storeErr := webhook.StorePending(j.cfg.Webhook.PendingDir, j.cfg.Webhook.URL, payload); storeErr != nil {
					log.Printf("[ERROR] Webhook: failed to store pending for retry: %v", storeErr)
				}
			}
		}
	}
}

func (j *Job) processBucket(ctx context.Context, bucket string, client r2.StorageClient, startTime time.Time) (int, int, []webhook.ImageEntry) {
	// 2. Load state
	statePath := j.statePathForBucket(bucket)
	currentState, err := state.LoadState(statePath)
	if err != nil {
		log.Printf("[ERROR] Failed to load state for bucket %s: %v", bucket, err)
		return 0, 1, nil
	}

	// 3. List objects since last processed time
	sinceStr := "beginning (full scan)"
	if !currentState.LastProcessedTime.IsZero() {
		sinceStr = currentState.LastProcessedTime.Format(time.RFC3339)
	}
	log.Printf("[INFO] Listing bucket: %s (since: %s)", bucket, sinceStr)
	keys, err := client.ListObjects(ctx, currentState.LastProcessedTime)
	if err != nil {
		log.Printf("[ERROR] Failed to list objects from R2 bucket %s: %v", bucket, err)
		return 0, 1, nil
	}

	log.Printf("[INFO] Found %d objects to check in bucket %s", len(keys), bucket)

	processedCount := 0
	failedCount := 0
	var converted []webhook.ImageEntry

	// 4. Process each image
	for _, key := range keys {
		// Skip if already webp
		if strings.HasSuffix(strings.ToLower(key), ".webp") {
			continue
		}

		// Check if extension is supported
		if !j.isSupportedExtension(key) {
			continue
		}

		log.Printf("[INFO] Processing image: %s", key)

		// Download
		data, err := client.DownloadImage(ctx, key)
		if err != nil {
			log.Printf("[ERROR] Failed to download image %s from bucket %s: %v", key, bucket, err)
			failedCount++
			continue
		}

		// Convert
		webpData, _, err := j.processor.Process(data, processor.ProcessOptions{})
		if err != nil {
			log.Printf("[ERROR] Failed to convert image %s: %v", key, err)
			failedCount++
			continue
		}

		// Upload with .webp extension
		destKey := j.changeExtensionToWebp(key)
		err = client.UploadImage(ctx, destKey, webpData, "image/webp")
		if err != nil {
			log.Printf("[ERROR] Failed to upload converted image %s to bucket %s: %v", destKey, bucket, err)
			failedCount++
			continue
		}

		log.Printf("[INFO] Successfully converted %s/%s to %s/%s", bucket, key, bucket, destKey)
		processedCount++
		converted = append(converted, webhook.ImageEntry{
			Bucket:      bucket,
			Source:      key,
			Destination: destKey,
		})

		// Delete original image
		/*
			if key != destKey {
				err = j.r2Client.DeleteObject(ctx, key)
				if err != nil {
					log.Printf("[WARN] Failed to delete original image %s: %v", key, err)
				} else {
					log.Printf("[INFO] Deleted original image: %s", key)
				}
			}
		*/

		// Note: We might want to keep track of the latest LastModified time from the objects
		// but since we don't have it here (ListObjects only returns keys),
		// we'll update based on the current time or some other logic.
		// For now, we'll update the state at the end.
	}

	// 5. Update state
	currentState.ProcessedCount = processedCount
	currentState.FailedCount = failedCount
	currentState.LastRunTime = startTime
	// Update last processed time to the start of this run
	// so next time we only look at images modified after this run started.
	currentState.UpdateLastProcessedTime(startTime)

	if err := state.SaveState(statePath, currentState); err != nil {
		log.Printf("[ERROR] Failed to save state for bucket %s: %v", bucket, err)
	}

	log.Printf("[INFO] Bucket %s processing completed. Processed: %d, Failed: %d",
		bucket, processedCount, failedCount)

	return processedCount, failedCount, converted
}

func (j *Job) statePathForBucket(bucket string) string {
	if len(j.cfg.R2.Buckets) <= 1 {
		return j.statePath
	}
	dir := filepath.Dir(j.statePath)
	return filepath.Join(dir, fmt.Sprintf("state-%s.json", sanitizeBucketForPath(bucket)))
}

func sanitizeBucketForPath(bucket string) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_")
	return replacer.Replace(bucket)
}

func (j *Job) bucketNames() []string {
	if len(j.cfg.R2.Buckets) > 0 {
		return j.cfg.R2.Buckets
	}
	if j.cfg.R2.Bucket != "" {
		return []string{j.cfg.R2.Bucket}
	}
	if len(j.r2Clients) == 1 {
		for bucket := range j.r2Clients {
			return []string{bucket}
		}
	}
	return nil
}

func (j *Job) isSupportedExtension(key string) bool {
	ext := strings.ToLower(filepath.Ext(key))
	if ext == "" {
		return false
	}
	// Remove dot
	ext = ext[1:]

	for _, format := range j.cfg.Conversion.Formats {
		if strings.ToLower(format) == ext || (ext == "jpeg" && format == "jpg") || (ext == "jpg" && format == "jpeg") {
			return true
		}
	}
	return false
}

func (j *Job) changeExtensionToWebp(key string) string {
	ext := filepath.Ext(key)
	if ext == "" {
		return key + ".webp"
	}
	return key[:len(key)-len(ext)] + ".webp"
}

func (j *Job) acquireLock() error {
	// Check if lock file exists and is old (stale lock prevention)
	info, err := os.Stat(j.lockPath)
	if err == nil {
		// Lock file exists. Check if it's older than 1 hour.
		if time.Since(info.ModTime()) > time.Hour {
			log.Println("[WARN] Found stale lock file, removing...")
			os.Remove(j.lockPath)
		} else {
			return fmt.Errorf("cron job is already running (lock file exists: %s)", j.lockPath)
		}
	}

	// Create directory if not exists
	dir := filepath.Dir(j.lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create lock file
	file, err := os.OpenFile(j.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			return fmt.Errorf("cron job is already running")
		}
		return err
	}
	file.Close()
	return nil
}

func (j *Job) releaseLock() {
	if err := os.Remove(j.lockPath); err != nil {
		log.Printf("[ERROR] Failed to release lock: %v", err)
	}
}
