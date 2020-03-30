package dto 

import (
	"time"
)

type Shit struct {
	Text      string    `json:"text"`
	ID        int       `gorm:"primary_key" json:"id,omitempty"`
	Timestamp time.Time `gorm:"type:DATE" json:"timestamp,omitempty"`
}


type Response struct {
	Status  ResponseStatus `json:"status"`
	Code    int            `json:"code,omitempty"`
	Data    interface{}    `json:"data,omitempty"`
	Message string         `json:"message,omitempty"`
}

type ResponseStatus string

const (
	Success ResponseStatus = "success"
	Fail    ResponseStatus = "fail"
	Error   ResponseStatus = "error"
)
