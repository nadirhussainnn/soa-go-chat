package models

type ContactRequest struct {
	ID          uint   `gorm:"primaryKey"`
	SenderID    uint   `gorm:"not null"`
	ReceiverID  uint   `gorm:"not null"`
	RequestTime int64  `gorm:"autoCreateTime"`
	Status      string `gorm:"default:pending"` // pending, accepted, rejected
}
