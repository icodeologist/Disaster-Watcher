package models

// May be we can get and generate the device token with expo thing
type DeviceTokenRequest struct {
	UserID uint   `json:"userId"`
	Token  string `json:"token"`
}
