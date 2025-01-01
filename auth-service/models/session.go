package models

import "github.com/google/uuid"

type Session struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null"`
	Token     string    `gorm:"uniqueIndex;not null"`
	CreatedAt int64     `gorm:"autoCreateTime"`
}
