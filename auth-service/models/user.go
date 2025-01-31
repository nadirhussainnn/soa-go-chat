// Defining the database tables, and related structs to store, get user related data
// Author: Nadir Hussain

package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"` // UUID as primary key
	Username string    `gorm:"uniqueIndex;not null"`
	Password string    `gorm:"not null"`
	Email    string    `gorm:"uniqueIndex;not null"`
}

type SenderDetails struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// BeforeCreate hook generates UUID before saving a new user
func (user *User) BeforeCreate(tx *gorm.DB) (err error) {
	user.ID = uuid.New()
	return
}
