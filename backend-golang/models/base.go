package models

import (
	"time"

	"github.com/google/uuid"
)

type ModelBase struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;"`
	TimeCreated time.Time  `json:"time_created"`
	TimeUpdated time.Time  `json:"time_updated"`
	TimeDeleted *time.Time `json:"time_deleted,omitempty" gorm:"index"`
}
