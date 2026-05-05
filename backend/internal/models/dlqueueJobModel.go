package models

import "time"

type DLQJob struct {
	ID             int64     `json:"id"`
	ErrorMessage   string    `json:"error_message"`
	FailedMsgJOBID int64     `json:"job_id"`
	CreatedAt      time.Time `json:"created_at"`
	WhereFailed    string    `json:"where_failed"`
}
