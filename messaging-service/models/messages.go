package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey"`
	SenderID       uuid.UUID      `gorm:"type:uuid;not null"`
	ReceiverID     uuid.UUID      `gorm:"type:uuid;not null"`
	Content        string         `json:"content"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	ContactDetails *SenderDetails `gorm:"-"`
}

type SenderDetails struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
