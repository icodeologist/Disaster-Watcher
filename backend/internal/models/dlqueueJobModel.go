package models

type DLQJob struct {
	ID int64
	// Why it failed
	ErrorInfo      string
	FailedMsgJOBID int64
}

type FailedInfo struct {
	Error error
}
