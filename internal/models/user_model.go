package models

type User struct {
	ID             uint     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserName       string   `json:"username" gorm:"unique;not null"`
	Email          string   `json:"email" gorm:"unique;not null"`
	Password       string   `json:"password"`
	Location       string   `json:"location"`
	Role           string   `json:"role"` // admin or user
	CachedLat      *float64 `json:"cachedlat" gorm:"column:cachedlat"`
	CachedLong     *float64 `json:"cachedlong" gorm:"column:cachedlong"`
	LocationCached bool     `json:"locationcached" gorm:"default:false"`
}

// this help during auth handlers
type AuthInput struct {
	Username    string `json:"username"`
	Email       string `json:"email"`
	Password    string `json:"password"`
	Location    string `json:"location"`
	PhoneNumber uint64 `json:"phonenumber"`
}
