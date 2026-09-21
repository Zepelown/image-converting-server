package webhook

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestBatchPayloadMarshal_NilImagesUsesEmptyArray(t *testing.T) {
	body, err := json.Marshal(BatchPayload{
		Event:          "batch.completed",
		ProcessedCount: 0,
		FailedCount:    0,
		Images:         nil,
	})
	if err != nil {
		t.Fatalf("marshal batch payload: %v", err)
	}

	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal batch payload: %v", err)
	}
	if got := string(payload["images"]); got != "[]" {
		t.Errorf("images = %s, want []", got)
	}
}

func TestSendBulk_EmptyURL_NoRequest(t *testing.T) {
	ctx := context.Background()
	payload := &BatchPayload{
		Event:          "batch.completed",
		ProcessedCount: 0,
		FailedCount:    0,
		Images:         nil,
	}
	err := SendBulk(ctx, "", payload, 10*time.Second)
	if err != nil {
		t.Errorf("SendBulk with empty URL should return nil, got: %v", err)
	}
}
