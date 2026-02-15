package webhook

import (
	"context"
	"testing"
	"time"
)

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
