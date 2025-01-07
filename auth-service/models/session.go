// Defining the database tables for sessions
// Author: Nadir Hussain

package models

import "github.com/google/uuid"

// Session model to store session related info for users
type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
	CreatedAt int64     `gorm:"autoCreateTime"`
}
