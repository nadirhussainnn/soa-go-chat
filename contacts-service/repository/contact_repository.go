// Repository to handle database operations needed for contacts and requests functionalities
// Author: Nadir Hussain

package repository

import (
	"contacts-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContactsRepository defines the interface for database operations related to contacts and requests.
type ContactsRepository interface {
	AcceptOrReject(contact *models.Contact) error
	AddContactRequest(req *models.ContactRequest) error
	GetContactsByUserID(userID uuid.UUID) ([]models.Contact, error)
	RemoveContact(senderID, receiverID string) error
	GetContactRequestByID(userID string) (*models.ContactRequest, error)
	DeleteRequest(req *models.ContactRequest) error
	GetContactRequestsByUserID(userID uuid.UUID) ([]models.ContactRequest, error)
}

// contactsRepository provides the implementation for ContactsRepository.
type contactsRepository struct {
	db *gorm.DB
}

// Initializes a new instance of contactsRepository.
// Parameters:
// - db: *gorm.DB - The database connection.
//
// Returns:
// - ContactsRepository: An instance of the ContactsRepository interface.

func NewContactsRepository(db *gorm.DB) ContactsRepository {
	return &contactsRepository{db: db}
}

// Creates a new contact in the database, marking the request as accepted.
// Parameters:
// - contact: *models.Contact - The contact information to be added.
//
// Returns:
// - error: An error if the operation fails, otherwise nil.

func (r *contactsRepository) AcceptOrReject(contact *models.Contact) error {
	return r.db.Create(contact).Error
}

// Adds a new contact request to the database.
// Parameters:
// - req: *models.ContactRequest - The contact request to be added.
//
// Returns:
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) AddContactRequest(req *models.ContactRequest) error {
	return r.db.Create(req).Error
}

// Fetches all contacts associated with a specific user ID.
// Parameters:
// - userID: uuid.UUID - The ID of the user whose contacts are to be fetched.
//
// Returns:
// - []models.Contact: A slice of contacts belonging to the user.
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) GetContactsByUserID(userID uuid.UUID) ([]models.Contact, error) {
	var contacts []models.Contact
	err := r.db.Where("user_id = ?", userID).Find(&contacts).Error
	return contacts, err
}

// Deletes a contact from the database by sender and receiver IDs.
// Parameters:
// - senderID: string - The ID of the sender.
// - receiverID: string - The ID of the receiver.
//
// Returns:
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) RemoveContact(senderID, receiverID string) error {
	return r.db.Where("user_id = ? AND contact_id = ?", senderID, receiverID).Delete(&models.Contact{}).Error
}

// Deletes a contact request from the database by its ID.
// Parameters:
// - req: *models.ContactRequest - The contact request to be deleted.
//
// Returns:
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) DeleteRequest(req *models.ContactRequest) error {
	return r.db.Where("id = ?", req.ID).Delete(&models.ContactRequest{}).Error
}

// Fetches all pending contact requests for a specific user ID.
// Parameters:
// - userID: uuid.UUID - The ID of the user whose contact requests are to be fetched.
//
// Returns:
// - []models.ContactRequest: A slice of pending contact requests for the user.
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) GetContactRequestsByUserID(userID uuid.UUID) ([]models.ContactRequest, error) {
	var contactRequests []models.ContactRequest
	// Filter requests where receiver_id matches and status is "pending"
	err := r.db.Where("receiver_id = ? AND status = ?", userID, "pending").Find(&contactRequests).Error
	return contactRequests, err
}

// Fetches a specific contact request by its ID.
// Parameters:
// - requestID: string - The ID of the contact request to be fetched.
//
// Returns:
// - *models.ContactRequest: The contact request if found, otherwise nil.
// - error: An error if the operation fails, otherwise nil.
func (r *contactsRepository) GetContactRequestByID(requestID string) (*models.ContactRequest, error) {
	var request models.ContactRequest
	err := r.db.Where("id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}
