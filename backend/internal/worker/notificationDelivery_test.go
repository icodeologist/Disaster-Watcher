package worker

import (
	"testing"
	"time"

	"github.com/icodeologist/disasterwatch/internal/models"
)

func TestSendRequestFromDeliveryUsesStoredSnapshot(t *testing.T) {
	delivery := models.NotificationDelivery{
		IdempotencyKey: "disaster-email/12/34",
		RecipientEmail: "person@example.com",
		EmailBody: models.EmailBody{
			Title:      "Flood warning",
			Location:   "River road",
			Precaution: "Move to higher ground",
		},
	}

	request := sendRequestFromDelivery(delivery)
	if request.IdempotencyKey != delivery.IdempotencyKey {
		t.Fatalf("idempotency key = %q, want %q", request.IdempotencyKey, delivery.IdempotencyKey)
	}
	if request.To != delivery.RecipientEmail {
		t.Fatalf("recipient = %q, want %q", request.To, delivery.RecipientEmail)
	}
	if request.Title != delivery.EmailBody.Title || request.Location != delivery.EmailBody.Location || request.Precaution != delivery.EmailBody.Precaution {
		t.Fatalf("request body = %+v, want stored body %+v", request, delivery.EmailBody)
	}
}

func TestRetryDelay(t *testing.T) {
	tests := []struct {
		attempts int
		want     time.Duration
	}{
		{attempts: 1, want: 2 * time.Second},
		{attempts: 2, want: 4 * time.Second},
		{attempts: 5, want: 32 * time.Second},
	}

	for _, test := range tests {
		if got := retryDelay(test.attempts); got != test.want {
			t.Errorf("retryDelay(%d) = %s, want %s", test.attempts, got, test.want)
		}
	}
}
