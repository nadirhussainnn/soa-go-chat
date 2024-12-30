package repository

import (
	"contacts-service/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContactsRepository interface {
	AcceptOrReject(contact *models.Contact) error
	AddContactRequest(req *models.ContactRequest) error
	GetContactsByUserID(userID uuid.UUID) ([]models.Contact, error)
	GetContactRequestByID(userID string) (*models.ContactRequest, error)
	UpdateContactRequest(req *models.ContactRequest) error
	GetContactRequestsByUserID(userID uuid.UUID) ([]models.ContactRequest, error)
}

type contactsRepository struct {
	db *gorm.DB
}

func NewContactsRepository(db *gorm.DB) ContactsRepository {
	return &contactsRepository{db: db}
}

func (r *contactsRepository) AcceptOrReject(contact *models.Contact) error {
	return r.db.Create(contact).Error
}

func (r *contactsRepository) AddContactRequest(req *models.ContactRequest) error {
	return r.db.Create(req).Error
}

func (r *contactsRepository) GetContactsByUserID(userID uuid.UUID) ([]models.Contact, error) {
	var contacts []models.Contact
	err := r.db.Where("user_id = ?", userID).Find(&contacts).Error
	return contacts, err
}

func (r *contactsRepository) UpdateContactRequest(req *models.ContactRequest) error {
	return r.db.Model(&models.ContactRequest{}).Where("id = ?", req.ID).Updates(req).Error
}

func (r *contactsRepository) GetContactRequestsByUserID(userID uuid.UUID) ([]models.ContactRequest, error) {
	var contactRequests []models.ContactRequest
	// Filter requests where receiver_id matches and status is "pending"
	err := r.db.Where("receiver_id = ? AND status = ?", userID, "pending").Find(&contactRequests).Error
	return contactRequests, err
}

func (r *contactsRepository) GetContactRequestByID(requestID string) (*models.ContactRequest, error) {
	var request models.ContactRequest
	err := r.db.Where("id = ?", requestID).First(&request).Error
	if err != nil {
		return nil, err
	}
	return &request, nil
}
