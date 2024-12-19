package models

type Contact struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null"`
	ContactID uint `gorm:"not null"`
}
