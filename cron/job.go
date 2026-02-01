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

	"github.com/robfig/cron/v3"
)

// Job represents the cron job scheduler
type Job struct {
	cron      *cron.Cron
	cfg       *config.Config
	r2Client  r2.StorageClient
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

	// 2. Load state
	currentState, err := state.LoadState(j.statePath)
	if err != nil {
		log.Printf("[ERROR] Failed to load state: %v", err)
		return
	}

	// 3. List objects since last processed time
	ctx := context.Background()
	keys, err := j.r2Client.ListObjects(ctx, currentState.LastProcessedTime)
	if err != nil {
		log.Printf("[ERROR] Failed to list objects from R2: %v", err)
		return
	}

	log.Printf("[INFO] Found %d objects to check", len(keys))

	processedCount := 0
	failedCount := 0

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
		data, err := j.r2Client.DownloadImage(ctx, key)
		if err != nil {
			log.Printf("[ERROR] Failed to download image %s: %v", key, err)
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
		err = j.r2Client.UploadImage(ctx, destKey, webpData, "image/webp")
		if err != nil {
			log.Printf("[ERROR] Failed to upload converted image %s: %v", destKey, err)
			failedCount++
			continue
		}

		log.Printf("[INFO] Successfully converted %s to %s", key, destKey)
		processedCount++

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

	if err := state.SaveState(j.statePath, currentState); err != nil {
		log.Printf("[ERROR] Failed to save state: %v", err)
	}

	log.Printf("[INFO] Cron job execution completed. Processed: %d, Failed: %d, Duration: %v",
		processedCount, failedCount, time.Since(startTime))
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
