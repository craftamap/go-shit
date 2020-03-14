package shit

import (
	"time"
)

type Shit struct {
	Text      string    `json:"text"`
	ID        int       `gorm:"primary_key" json:"id,omitempty"`
	Timestamp time.Time `gorm:"type:DATE" json:"timestamp,omitempty"`
}
