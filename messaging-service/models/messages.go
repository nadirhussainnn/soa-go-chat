package models

import (
	"time"

	"github.com/google/uuid"
)

type Message struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SenderID     uuid.UUID `gorm:"type:uuid;not null" json:"sender_id"`
	ReceiverID   uuid.UUID `gorm:"type:uuid;not null" json:"receiver_id"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	MessageType  string    `gorm:"type:varchar(50);not null;default:'text'" json:"message_type"` // 'text', 'file'
	FilePath     string    `gorm:"type:text" json:"file_path"`                                   // Path to the file on the server
	FileName     string    `gorm:"type:text" json:"file_name"`                                   // Name of the file
	FileMimeType string    `gorm:"type:varchar(100)" json:"file_mime_type"`
}
