package repository

import (
	"messaging-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MessagesRepository interface {
	SendMessage(req *models.Message) error
	GetMessagesByUserID(userID uuid.UUID) ([]models.Message, error)
}

type messageRepository struct {
	db *gorm.DB
}

func NewMessagesRepository(db *gorm.DB) MessagesRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) SendMessage(req *models.Message) error {
	return r.db.Create(req).Error
}

func (r *messageRepository) GetMessagesByUserID(userID uuid.UUID) ([]models.Message, error) {
	var contacts []models.Message
	err := r.db.Where("user_id = ?", userID).Find(&contacts).Error
	return contacts, err
}
