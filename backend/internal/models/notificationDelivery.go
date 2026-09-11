package models

import "time"

const (
	DeliveryStatusPending    = "pending"
	DeliveryStatusProcessing = "processing"
	DeliveryStatusRetrying   = "retrying"
	DeliveryStatusSuccess    = "success"
	DeliveryStatusFailed     = "failed"
)

// NotificationDelivery is the durable record for one email sent to one user.
// The recipient and body are snapshots: retries must not rebuild them from data
// that may have changed since the delivery was created.
type NotificationDelivery struct {
	ID             uint   `gorm:"primaryKey"`
	IdempotencyKey string `gorm:"uniqueIndex;not null;size:255"`
	JobID          int64  `gorm:"uniqueIndex:idx_job_user;not null"`
	UserID         uint   `gorm:"uniqueIndex:idx_job_user;not null"`

	RecipientEmail string    `gorm:"not null;size:320"`
	EmailBody      EmailBody `gorm:"embedded;embeddedPrefix:email_"`

	Status              string `gorm:"index;not null"`
	Attempts            int
	ProcessingStartedAt *time.Time `gorm:"index"`
	NextAttemptAt       *time.Time `gorm:"index"`
	ProviderMessageID   string     `gorm:"size:255"`
	SentAt              *time.Time
	FailedAt            *time.Time
	LastError           string `gorm:"type:text"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
