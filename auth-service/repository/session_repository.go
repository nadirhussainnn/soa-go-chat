// Repository to handle database operations needed for session functionalities
// Author: Nadir Hussain

package repository

import (
	"auth-service/models"

	"gorm.io/gorm"
)

// Interface to define functions that are implemented in code below
type SessionRepository interface {
	CreateSession(session *models.Session) error
	DeleteSession(token string) error
	GetSessionByID(sessionID string) (*models.Session, error) // Fix: Match the implementation
}

// Provides a concrete implementation of the SessionRepository interface.
type sessionRepository struct {
	db *gorm.DB
}

// Initializes a new session repository with the provided GORM database instance.
// Params:
//   - db: A pointer to the GORM database instance.
//
// Returns:
//   - SessionRepository: An instance of the SessionRepository interface with database connection initialized.
func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

// CreateSession creates a new session for a user. Any existing session for the same user is deleted before creating a new session.
// Params:
//   - session: A pointer to the Session object containing session details to be created.
//
// Returns:
//   - error: An error object if the operation fails; otherwise, nil.
func (r *sessionRepository) CreateSession(session *models.Session) error {
	err := r.db.Where("user_id = ?", session.UserID).Delete(&models.Session{}).Error
	if err != nil {
		return err
	}
	return r.db.Create(session).Error
}

// Fetches a session from the database using the session ID.
// Params:
//   - sessionID: A string representing the unique ID of the session.
//
// Returns:
//   - *models.Session: A pointer to the Session object if found.
//   - error: An error object if the operation fails or the session is not found.
func (r *sessionRepository) GetSessionByID(sessionID string) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("id = ?", sessionID).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) DeleteSession(token string) error {
	return r.db.Where("token = ?", token).Delete(&models.Session{}).Error
}
