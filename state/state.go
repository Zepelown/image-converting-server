package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State represents the current status of the image conversion process.
type State struct {
	LastProcessedTime time.Time `json:"last_processed_time"`
	LastRunTime       time.Time `json:"last_run_time"`
	ProcessedCount    int       `json:"processed_count"`
	FailedCount       int       `json:"failed_count"`
}

// NewState creates a new State with default values.
func NewState() *State {
	return &State{
		LastProcessedTime: time.Now(),
		LastRunTime:       time.Now(),
		ProcessedCount:    0,
		FailedCount:       0,
	}
}

// LoadState loads the state from a JSON file.
// If the file does not exist, it returns a new initial state.
func LoadState(filePath string) (*State, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// If file doesn't exist, return a new state
			return NewState(), nil
		}
		return nil, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}

	return &state, nil
}

// SaveState saves the current state to a JSON file.
func SaveState(filePath string, state *State) error {
	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	// Write to a temporary file first and then rename to ensure atomicity
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}

	return os.Rename(tmpFile, filePath)
}

// UpdateLastProcessedTime updates the last processed time to the given time.
func (s *State) UpdateLastProcessedTime(t time.Time) {
	s.LastProcessedTime = t
}

// UpdateLastRunTime updates the last run time to the current time.
func (s *State) UpdateLastRunTime() {
	s.LastRunTime = time.Now()
}
