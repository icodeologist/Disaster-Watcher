package models

import (
	"time"
)

type ProcessedNotification struct {
	ID             uint   `gorm:"primaryKey"`
	IdempotencyKey string `gorm:"uniqueIndex;size:255"`
	JobID          int64  `gorm:"uniqueIndex:idx_job_user"`
	UserID         uint   `gorm:"uniqueIndex:idx_job_user"`
	Status         string `gorm:"index"` // "pending", "success", "failed"
	Attempts       int
	SentAt         *time.Time
	FailedAt       *time.Time
	LastError      string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
