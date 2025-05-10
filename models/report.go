package models

import (
	"time"
)

type Report struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Type        string    `json:"type"` // flood or animals or landslide or rain or electricity outrage
	Description string    `json:"description"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	TimeStamp   time.Time `json:"timestamp"`
	Status      string    `json:"status"` // solved or active
}
