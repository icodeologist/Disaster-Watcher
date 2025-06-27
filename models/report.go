package models

import (
	pb "github.com/icodeologist/grpc-proto"
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
	ID             uint     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserName       string   `json:"username" gorm:"unique;not null"`
	Email          string   `json:"email" gorm:"unique;not null"`
	Password       string   `json:"password"`
	PhoneNumber    uint64   `json:"phonenumber"`
	Location       string   `json:"location"`
	CachedLat      *float64 `json:"cachedlat" gorm:"column:cachedlat"`
	CachedLong     *float64 `json:"cachedlong" gorm:"column:cachedlong"`
	LocationCached bool     `json:"locationcached" gorm:"default:false"`
}

type Location struct {
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`
}
type AuthInput struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Location    string `json:"location"`
	PhoneNumber uint64 `json:"phonenumber"`
}

// geocoding response from nominatim api
type GeocodingResult struct {
	Latitude  string `json:"lat"`
	Longitude string `json:"lon"`
	PlaceName string `json:"placename"`
}

type PasswordResetToken struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Token     string    `json:"token" gorm:"uniqueIndex;not null"`
	UserID    uint      `json:"user_id" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
	Used      bool      `json:"used" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
}

type PasswordResetRequest struct {
	Token             string `json:"token"`
	NewPassword       string `json:"new-password"`
	ReTypeNewPassword string `json:"retype-newpassword"`
}

// The affected users are here
type NotificationEvent struct {
	UserID           uint
	User             User
	Action           string
	Message          string
	Timestamp        time.Time
	NotificationType pb.Notificationtype
}

type NotificationEvent2 struct {
	User             string
	Action           string
	Message          string
	TimeStamp        time.Time
	NotificationType pb.Notificationtype
}
