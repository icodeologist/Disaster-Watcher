package emailservice

import (
	"context"
	"errors"
	"testing"
)

func TestSendEmailRejectsMissingIdempotencyKey(t *testing.T) {
	err := SendEmail(context.Background(), SendRequest{To: "person@example.com"})
	if err == nil || err.Error() != "missing email idempotency key" {
		t.Fatalf("SendEmail() error = %v, want missing idempotency key", err)
	}
}

func TestSendEmailHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := SendEmail(ctx, SendRequest{IdempotencyKey: "delivery/1", To: "person@example.com"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("SendEmail() error = %v, want context.Canceled", err)
	}
}
