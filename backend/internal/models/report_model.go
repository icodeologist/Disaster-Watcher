package models

import "time"

type Location struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}

type Report struct {
	ID               uint      `json:"id" gorm:"primaryKey"`
	UserId           uint      `json:"userid"`
	User             User      `gorm:"foreignKey:UserId"`
	Title            string    `json:"title"`
	Category         string    `json:"category" gorm:"not null"`    // flood or animals or landslide or rain or electricity outrage
	Description      string    `json:"description" gorm:"not null"` // valid desc == and frequent similary reports at the same area or neaby -- Approve
	Location         string    `json:"location"`
	Priority         string    `json:"priority" gorm:"not null"`         // critical , high, medium, low
	Status           string    `json:"status" gorm:"default:unverified"` // immediate , review, verified, false --  admin checks this or smart fileter ?
	Upvote           int       `json:"upvote" gorm:"default:0"`
	Downvote         int       `json:"downvote" gorm:"default:0"`
	CachedLat        *float64  `json:"latitude"`
	CachedLong       *float64  `json:"longitude"`
	ISLocationCached bool      `json:"is-cached"`
	CreatedAt        time.Time `json:"created_at"`
}
