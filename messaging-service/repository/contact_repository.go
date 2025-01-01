package repository

import (
	"messaging-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessageRepository interface {
	CreateNewMessage(req *models.Message) error
	GetMessagesByUserID(userID uuid.UUID) ([]models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewContactsRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) CreateNewMessage(req *models.Message) error {
	return r.db.Create(req).Error
}

func (r *messageRepository) GetMessagesByUserID(userID uuid.UUID) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("sender_id = ? OR receiver_id = ?", userID, userID).Find(&messages).Error
	return messages, err
}
