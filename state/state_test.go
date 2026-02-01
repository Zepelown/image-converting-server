package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStateLoadSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	statePath := filepath.Join(tempDir, "state.json")

	// Test LoadState for non-existent file
	s1, err := LoadState(statePath)
	if err != nil {
		t.Errorf("LoadState failed: %v", err)
	}
	if s1 == nil {
		t.Fatal("LoadState returned nil")
	}

	// Modify state and save
	now := time.Now().Round(time.Second) // Round to avoid precision issues in JSON
	s1.LastProcessedTime = now
	s1.ProcessedCount = 10
	s1.FailedCount = 2

	if err := SaveState(statePath, s1); err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load again and compare
	s2, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}

	if !s2.LastProcessedTime.Equal(s1.LastProcessedTime) {
		t.Errorf("LastProcessedTime mismatch: expected %v, got %v", s1.LastProcessedTime, s2.LastProcessedTime)
	}
	if s2.ProcessedCount != s1.ProcessedCount {
		t.Errorf("ProcessedCount mismatch: expected %d, got %d", s1.ProcessedCount, s2.ProcessedCount)
	}
	if s2.FailedCount != s1.FailedCount {
		t.Errorf("FailedCount mismatch: expected %d, got %d", s1.FailedCount, s2.FailedCount)
	}
}

func TestNewState(t *testing.T) {
	s := NewState()
	if s == nil {
		t.Fatal("NewState returned nil")
	}
	if s.ProcessedCount != 0 {
		t.Errorf("Expected ProcessedCount 0, got %d", s.ProcessedCount)
	}
	if s.FailedCount != 0 {
		t.Errorf("Expected FailedCount 0, got %d", s.FailedCount)
	}
	if s.LastProcessedTime.IsZero() {
		t.Error("LastProcessedTime should not be zero")
	}
}

func TestUpdateFunctions(t *testing.T) {
	s := NewState()

	newProcessedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	s.UpdateLastProcessedTime(newProcessedTime)

	if !s.LastProcessedTime.Equal(newProcessedTime) {
		t.Errorf("UpdateLastProcessedTime failed: expected %v, got %v", newProcessedTime, s.LastProcessedTime)
	}

	oldRunTime := s.LastRunTime
	time.Sleep(10 * time.Millisecond)
	s.UpdateLastRunTime()

	if !s.LastRunTime.After(oldRunTime) {
		t.Errorf("UpdateLastRunTime failed: new time %v should be after old time %v", s.LastRunTime, oldRunTime)
	}
}
