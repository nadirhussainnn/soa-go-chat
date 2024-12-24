package models

import (
	"time"

	"github.com/google/uuid"
)

type Contact struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	ContactID uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

type ContactRequest struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	SenderID   uuid.UUID `gorm:"type:uuid;not null"`
	ReceiverID uuid.UUID `gorm:"type:uuid;not null"`
	Status     string    `gorm:"default:pending"` // pending, accepted, rejected
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}
