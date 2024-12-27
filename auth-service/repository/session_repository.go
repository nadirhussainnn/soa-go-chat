package repository

import (
	"auth-service/models"

	"gorm.io/gorm"
)

type SessionRepository interface {
	CreateSession(session *models.Session) error
	DeleteSession(token string) error
	GetSessionByID(sessionID string) (*models.Session, error) // Fix: Match the implementation
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) CreateSession(session *models.Session) error {
	err := r.db.Where("user_id = ?", session.UserID).Delete(&models.Session{}).Error
	if err != nil {
		return err
	}
	return r.db.Create(session).Error
}

// GetSessionByID fetches a session by its ID
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
