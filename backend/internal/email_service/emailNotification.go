package emailservice

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
)

// SendRequest contains only the values an email provider needs.
type SendRequest struct {
	IdempotencyKey string
	To             string
	Title          string
	Location       string
	Precaution     string
}

func SendEmail(ctx context.Context, request SendRequest) error {
	// now we pass this to provider and it should handle the rest
	// if it has already seen this give us the same response exit from here
	// if not just actually send the email
	if err := ctx.Err(); err != nil {
		return err
	}
	if request.IdempotencyKey == "" {
		return errors.New("missing email idempotency key")
	}
	if request.To == "" {
		return errors.New("missing email recipient")
	}
	if rand.IntN(2) == 0 {
		return errors.New("simulated email delivery failure")
	}

	slog.Info("Simulated email sent successfully", "email", request.To)
	return nil
}
