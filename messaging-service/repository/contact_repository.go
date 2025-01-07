package repository

import (
	"messaging-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Defines the methods for database operations related to messages.
type MessageRepository interface {
	CreateNewMessage(req *models.Message) error
	GetMessagesByUserID(userID, contactID uuid.UUID) ([]models.Message, error)
	GetMessageByID(messageId uuid.UUID) (models.Message, error)
}

// Implementation of MessageRepository that uses GORM for database operations.
type messageRepository struct {
	db *gorm.DB
}

// Initializes a new instance of MessageRepository.
//
// Parameters:
// - db: *gorm.DB - The GORM database instance.
// Returns:
// - MessageRepository: The initialized message repository.
func NewContactsRepository(db *gorm.DB) MessageRepository {
	return &messageRepository{db: db}
}

// Adds a new message record to the database.
// Parameters:
// - req: *models.Message - The message to be added to the database.
// Returns:
// - error: Returns an error if the message could not be created.

func (r *messageRepository) CreateNewMessage(req *models.Message) error {
	return r.db.Create(req).Error
}

// Retrieves a specific message from the database by its ID.
// Parameters:
// - messageId: uuid.UUID - The unique identifier of the message to retrieve.
// Returns:
// - models.Message: The retrieved message.
// - error: Returns an error if the message could not be found.
func (r *messageRepository) GetMessageByID(messageId uuid.UUID) (models.Message, error) {
	var message models.Message
	err := r.db.Where("id = ? ", messageId).First(&message).Error
	return message, err
}

// Fetches all messages exchanged between two users.
// Parameters:
// - userID: uuid.UUID - The ID of the logged-in user.
// - contactID: uuid.UUID - The ID of the contact.
// Returns:
// - []models.Message: A list of messages exchanged between the user and the contact.
// - error: Returns an error if the messages could not be fetched.
func (r *messageRepository) GetMessagesByUserID(userID, contactID uuid.UUID) ([]models.Message, error) {
	var messages []models.Message
	err := r.db.Where("(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		userID, contactID, contactID, userID).Find(&messages).Error
	return messages, err
}
