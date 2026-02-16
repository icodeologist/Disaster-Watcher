package models

type Error struct {
	ErrorCode    string `json:"code"`              // client readable error codes.
	Message      string `json:"message,omitempty"` // just a small brief message on what is wrong or why this happend?
	ErrorDetails any    `json:"details,omitempty"` // computer specific errors that occurred => err.Error()
}

type SuccessResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Message string `json:"message,omitempty"`
}

type ErrorResponse struct {
	Success bool  `json:"success"`
	Error   Error `json:"error"`
}

type UserREGISTERRDataResponse struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Location    string     `json:"location"`
	LatANDLongs CachedCord `json:"co-ordinates"`
}

type CachedCord struct {
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"Longitude"`
}

type ReportCreatedResponse struct {
	ID          uint   `json:"id"`
	CreatedBy   string `json:"create_by"`
	Category    string `json:"category"`
	Description string `json:"Description"`
	Location    string `json:"location"`
	Status      string `json:"status"`
}

type UserAccountInfoResponse struct {
	UserName     string `json:"username"`
	UserEmail    string `json:"email"`
	UserLocation string `json:"location"`
}
