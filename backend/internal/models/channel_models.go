package models

type ReportMessage struct {
	JobID  int64
	Report Report
}

// NotificationDeliveryMessage is only a wake-up signal. The database row is
// the source of truth for the delivery's current state and email contents.
type NotificationDeliveryMessage struct {
	DeliveryID uint
}

type VerificationMessage struct {
	JobID int64
}
