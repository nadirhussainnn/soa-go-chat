package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	SenderID   uuid.UUID `gorm:"type:uuid;not null"`
	ReceiverID uuid.UUID `gorm:"type:uuid;not null"`
	Content    string    `gorm:"type:text;not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}
