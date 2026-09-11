package models

type EmailBody struct {
	Title      string
	Location   string
	Precaution string
}

type EmailModel struct {
	ID             uint      `gorm:"primaryKey"`
	IdempotencyKey string    `gorm:"uniqueIndex;not null;size:255"`
	Email          string    `gorm:"not null;size:320"`
	EmailBody      EmailBody `gorm:"embedded;embeddedPrefix:body_"`
}
