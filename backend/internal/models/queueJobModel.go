package models

import (
	"time"
)

// adding byte coz postgres tables has json bites. paster
// TODO: understand this
type Jobs struct {
	Id         int64
	Status     string
	Payload    []byte
	Created_at time.Time
	Started_at time.Time
	Attempts   int32
}

type PayloadData struct {
	ReportID uint
	UserID   uint
}
