package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null"`
	ContactID      uuid.UUID      `gorm:"type:uuid;not null"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	ContactDetails *SenderDetails `gorm:"-"`
}

type ContactRequest struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	SenderID           uuid.UUID      `gorm:"type:uuid;not null" json:"sender_id"`
	ReceiverID         uuid.UUID      `gorm:"type:uuid;not null" json:"receiver_id"`
	Status             string         `gorm:"default:pending" json:"status"` // pending, accepted, rejected
	CreatedAt          time.Time      `gorm:"autoCreateTime" json:"created_at"`
	SenderDetails      *SenderDetails `gorm:"-" json:"sender_details,omitempty"`
	TargetUserDetails  *SenderDetails `gorm:"-" json:"target_user_details,omitempty"`
	CreatedAtFormatted string         `gorm:"-" json:"created_at_formatted,omitempty"`
}

type SenderDetails struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
