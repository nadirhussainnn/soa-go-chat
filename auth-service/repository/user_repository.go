// Repository to handle database operations for user management.
// Author: Nadir Hussain

package repository

import (
	"auth-service/models"

	"gorm.io/gorm"
)

// Defines the interface for user-related database operations.
type UserRepository interface {
	CreateUser(user *models.User) error
	GetUserByUsername(username string) (*models.User, error)
	GetUserByID(id string) (*models.User, error)
	UpdateUser(user *models.User) error
	SearchUser(query, excludeUserID string) ([]models.User, error)
}

// Provides a concrete implementation of the UserRepository interface.
type userRepository struct {
	db *gorm.DB
}

// Initializes a new user repository with the provided GORM database instance.
// Params:
//   - db: A pointer to the GORM database instance.
//
// Returns:
//   - UserRepository: An instance of the UserRepository interface with database connection initialized.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Adds a new user to the database.
// Params:
//   - user: A pointer to the User object containing user details to be created.
//
// Returns:
//   - error: An error object if the operation fails; otherwise, nil.
func (r *userRepository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

// SearchUser searches for users based on a query, excluding a specific user by their ID.
// Params:
//   - query: A string used to search for users (e.g., by username).
//   - excludeUserID: A string representing the ID of the user to exclude from the search.
// Returns:
//   - []models.User: A slice of User objects matching the search criteria.
//   - error: An error object if the operation fails; otherwise, nil.

func (r *userRepository) SearchUser(query, excludeUserID string) ([]models.User, error) {
	var users []models.User
	err := r.db.Raw(`SELECT * FROM users WHERE username LIKE ? AND id != ?`, "%"+query+"%", excludeUserID).Scan(&users).Error
	return users, err
}

//	Retrieves a user from the database based on their username or email.
//
// Params:
//   - username: A string representing the username or email of the user to retrieve.
//
// Returns:
//   - *models.User: A pointer to the User object if found.
//   - error: An error object if the operation fails or the user is not found.
func (r *userRepository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ? OR email = ?", username, username).First(&user).Error
	return &user, err
}

// Retrieves a user from the database using their unique ID.
// Params:
//   - id: A string representing the unique ID of the user.
//
// Returns:
//   - *models.User: A pointer to the User object if found.
//   - error: An error object if the operation fails or the user is not found.
func (r *userRepository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	return &user, err
}

// Updates the details of an existing user in the database.
// Params:
//   - user: A pointer to the User object containing updated user details.
// Returns:
//   - error: An error object if the operation fails; otherwise, nil.

func (r *userRepository) UpdateUser(user *models.User) error {
	return r.db.Save(user).Error
}
