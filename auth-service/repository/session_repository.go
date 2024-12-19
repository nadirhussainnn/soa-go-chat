package repository

import (
	"auth-service/models"

	"gorm.io/gorm"
)

type SessionRepository interface {
	CreateSession(session *models.Session) error
	GetSessionByToken(token string) (*models.Session, error)
	DeleteSession(token string) error
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

func (r *sessionRepository) GetSessionByToken(token string) (*models.Session, error) {
	var session models.Session
	err := r.db.Where("token = ?", token).First(&session).Error
	return &session, err
}

func (r *sessionRepository) DeleteSession(token string) error {
	return r.db.Where("token = ?", token).Delete(&models.Session{}).Error
}
