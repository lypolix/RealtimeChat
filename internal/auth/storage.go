package auth 

import (
	"RealtimeChat/internal/auth/models"

)

type Storage interface {
    GetUserByEmail(email string) (*models.User, error)
    CreateUser(user *models.User) error
}