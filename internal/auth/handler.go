package auth

import (
	"encoding/json"
	"net/http"
	"time"
)

// swagger:model Credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// swagger:model AuthResponse
type AuthResponse struct {
	Token string `json:"token"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// swagger:model User
type User struct {
    ID        string    `json:"id" example:"123e4567-e89b-12d3-a456-426614174000"`
    Email     string    `json:"email" example:"user@example.com"`
    CreatedAt time.Time `json:"created_at" example:"2023-07-22T14:12:00Z"`
    UpdatedAt time.Time `json:"updated_at" example:"2023-07-25T18:34:00Z"`
}


// @Summary Вход пользователя
// @Description Проверяет email/пароль и возвращает JWT токен
// @Tags auth
// @Accept json
// @Produce json
// @Param body body Credentials true "User credentials"
// @Success 200 {object} AuthResponse
// @Failure 401 {string} string "Invalid credentials"
// @Router /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	token, err := h.service.Login(creds.Email, creds.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

// @Summary Регистрация пользователя
// @Description Создаёт нового пользователя
// @Tags auth
// @Accept json
// @Produce json
// @Param body body Credentials true "User credentials"
// @Success 201 {string} string "User created"
// @Failure 400 {string} string "Invalid request body"
// @Router /register [post]
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    var req Credentials
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }

    err := h.service.Register(req.Email, req.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("User created"))
}
