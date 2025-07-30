package auth

import (
	"encoding/json"
	"net/http"
)

// Credentials представляет структуру для входящих данных авторизации
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse представляет структуру ответа с токеном
type AuthResponse struct {
	Token string `json:"token"`
}

// Handler обрабатывает HTTP-запросы для аутентификации
type Handler struct {
	service *Service
}

// NewHandler создает новый экземпляр Handler
func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

// Login обрабатывает запрос на вход в систему
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Проверяем метод запроса
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Декодируем тело запроса
	var creds Credentials
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Вызываем сервис для аутентификации
	token, err := h.service.Login(creds.Email, creds.Password)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Формируем успешный ответ
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(AuthResponse{Token: token})
}

// Register обрабатывает запрос на создание нового пользователя
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
    // Вызовем сервис регистрации
    err := h.service.Register(req.Email, req.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("User created"))
}
