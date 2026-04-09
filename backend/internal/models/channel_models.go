package models

import (
	"time"
)

type ReportMessage struct {
	JobID  int64
	Report Report
}

type AffectedUsersMessage struct {
	JobID  int64
	UserID uint
}

type FailedEmailMessage struct {
	JobID        int64
	User         User
	ErrorMessage error
	RetryAttempt int
	RetryDelay   time.Duration
}

type VerificationMessage struct {
	JobID int64
}
