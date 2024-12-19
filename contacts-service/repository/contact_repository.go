package repository

import (
	"auth-service/models"
	"time"

	"gorm.io/gorm"
)

type ContactRepository struct {
	db *gorm.DB
}

func NewContactRepository(db *gorm.DB) *ContactRepository {
	return &ContactRepository{db: db}
}

// Fetch all available users (excluding current user's contacts)
func (repo *ContactRepository) GetAvailableUsers(userID uint) ([]models.User, error) {

	var users []models.User
	err := repo.db.Raw(`
		SELECT * FROM users WHERE id NOT IN 
		(SELECT contact_id FROM contacts WHERE user_id = ?) AND id != ?
	`, userID, userID).Scan(&users).Error
	return users, err
}

// Fetch current user's contacts
func (repo *ContactRepository) GetUserContacts(userID uint) ([]models.User, error) {
	var contacts []models.User
	err := repo.db.Raw(`
		SELECT * FROM users WHERE id IN 
		(SELECT contact_id FROM contacts WHERE user_id = ?)
	`, userID).Scan(&contacts).Error
	return contacts, err
}

// Search for users by username
func (repo *ContactRepository) SearchUsers(query string, userID uint) ([]models.User, error) {
	var users []models.User
	err := repo.db.Raw(`
		SELECT * FROM users WHERE username LIKE ? AND id NOT IN 
		(SELECT contact_id FROM contacts WHERE user_id = ?) AND id != ?
	`, "%"+query+"%", userID, userID).Scan(&users).Error
	return users, err
}

// Send a contact request
func (repo *ContactRepository) SendContactRequest(senderID, receiverID uint) error {
	request := models.ContactRequest{
		SenderID:    senderID,
		ReceiverID:  receiverID,
		RequestTime: time.Now().Unix(),
	}
	return repo.db.Create(&request).Error
}

// Remove a contact
func (repo *ContactRepository) RemoveContact(userID, contactID uint) error {
	return repo.db.Where("user_id = ? AND contact_id = ?", userID, contactID).Delete(&models.Contact{}).Error
}

func (repo *ContactRepository) GetPendingRequests(userID uint) ([]models.ContactRequest, error) {
	var requests []models.ContactRequest
	err := repo.db.Where("receiver_id = ? AND status = ?", userID, "pending").Find(&requests).Error
	return requests, err
}

func (repo *ContactRepository) UpdateRequestStatus(requestID uint, status string) error {
	return repo.db.Model(&models.ContactRequest{}).Where("id = ?", requestID).Update("status", status).Error
}
