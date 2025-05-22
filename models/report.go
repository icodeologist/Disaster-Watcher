package models

import (
	"time"
)

type Report struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	UserId       uint      `json:"userid"`
	User         User      `gorm:"foreignKey:UserId"`
	Type         string    `json:"type"` // flood or animals or landslide or rain or electricity outrage
	Description  string    `json:"description"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Status       string    `json:"status"` // solved or active
	Created_time time.Time `json:"createdtime"`
	Updated_time time.Time `json:"updatedtime"`
}

type User struct {
	ID       uint   `json:"id" gorm:"primaryKey;autoIncrement"`
	UserName string `json:"username" gorm:"unique;not null"`
	Email    string `json:"email" gorm:"unique;not null"`
	Password string `json:"password"`
}

type AuthInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}
