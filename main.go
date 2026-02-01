package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"image-converting-server/api"
	"image-converting-server/config"
	"image-converting-server/cron"
	"image-converting-server/processor"
	"image-converting-server/r2"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("[FATAL] Failed to load configuration: %v", err)
	}

	// 2. Initialize R2 client
	ctx := context.Background()
	storageClient, err := r2.NewClient(ctx, &cfg.R2)
	if err != nil {
		log.Fatalf("[FATAL] Failed to initialize R2 client: %v", err)
	}

	// Test R2 connection
	if err := storageClient.TestConnection(ctx); err != nil {
		log.Printf("[WARN] R2 connection test failed: %v. Please check your credentials.", err)
	} else {
		log.Println("[INFO] Successfully connected to R2 bucket")
	}

	// 3. Initialize Image Processor
	proc := processor.NewProcessor(*cfg)

	// 4. Initialize Cron Job
	statePath := "data/state.json"
	cronJob := cron.NewJob(cfg, storageClient, proc, statePath)
	if err := cronJob.Start(); err != nil {
		log.Fatalf("[FATAL] Failed to start cron job: %v", err)
	}
	defer cronJob.Stop()

	// 5. Setup HTTP Router
	handler := api.NewHandler(storageClient, proc, cfg)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler.HandleIndex)
	mux.HandleFunc("/health", handler.HandleHealth)
	mux.HandleFunc("/api/convert", handler.HandleConvert)

	// 6. Start HTTP Server
	port := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         port,
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.TimeoutSeconds) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.TimeoutSeconds) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("[INFO] Server starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] Failed to start server: %v", err)
		}
	}()

	// 7. Implement Graceful Shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Wait for termination signal
	<-stop

	log.Println("[INFO] Shutting down server...")

	// Create a context with timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] Server forced to shutdown: %v", err)
	}

	log.Println("[INFO] Server exiting properly")
}
