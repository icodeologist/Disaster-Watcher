package models

type DLQJob struct {
	ID int64
	// Why it failed
	Error         string
	FailedMsgInfo int
}

type FailedInfo struct {
	AttemptedTime int
}
