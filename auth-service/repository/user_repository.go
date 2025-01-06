package repository

import (
	"auth-service/models"

	"gorm.io/gorm"
)

type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateUser(user *models.User) error
	SearchUser(query, excludeUserID string) ([]models.User, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *userRepository) SearchUser(query, excludeUserID string) ([]models.User, error) {
	var users []models.User
	err := r.db.Raw(`SELECT * FROM users WHERE username LIKE ? AND id != ?`, "%"+query+"%", excludeUserID).Scan(&users).Error
	return users, err
}

func (r *userRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? OR email = ?", username, username).First(&user).Error
	return &user, err
}

func (r *userRepository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

func (r *userRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}
