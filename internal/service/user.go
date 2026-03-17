// Package service implementa los casos de uso y reglas del negocio.
package service

import (
	"fmt"
	"strings"
	"time"

	"tienda-go/internal/domain"
	"tienda-go/internal/platform/ids"
	"tienda-go/internal/security"
	"tienda-go/internal/store"
)

// UserService concentra la logica de alta y consulta de usuarios.
type UserService struct {
	store store.Store
}

// CreateUserInput define los datos requeridos para registrar una cuenta.
type CreateUserInput struct {
	Name     string
	Email    string
	Password string
	Role     domain.Role
}

// NewUserService construye el servicio de usuarios.
func NewUserService(store store.Store) *UserService {
	return &UserService{store: store}
}

// CreateUser valida la entrada, genera salt/hash y persiste el usuario.
func (s *UserService) CreateUser(input CreateUserInput) (domain.User, error) {
	name := strings.TrimSpace(input.Name)
	email := strings.ToLower(strings.TrimSpace(input.Email))

	if name == "" || email == "" || strings.TrimSpace(input.Password) == "" {
		return domain.User{}, fmt.Errorf("%w: name, email and password are required", domain.ErrValidation)
	}

	if len(input.Password) < 8 {
		return domain.User{}, fmt.Errorf("%w: password must contain at least 8 characters", domain.ErrValidation)
	}

	if !isValidRole(input.Role) {
		return domain.User{}, fmt.Errorf("%w: unsupported role %q", domain.ErrValidation, input.Role)
	}

	salt, err := security.NewSalt()
	if err != nil {
		return domain.User{}, err
	}

	now := time.Now().UTC()
	user := domain.User{
		ID:           ids.New("usr"),
		Name:         name,
		Email:        email,
		Role:         input.Role,
		PasswordSalt: salt,
		PasswordHash: security.HashPassword(input.Password, salt),
		Active:       true,
		CreatedAt:    now,
	}

	if err := s.store.CreateUser(user); err != nil {
		return domain.User{}, err
	}

	return user, nil
}

// GetUser busca un usuario por ID.
func (s *UserService) GetUser(id string) (domain.User, error) {
	return s.store.GetUserByID(strings.TrimSpace(id))
}

// ListUsers devuelve todos los usuarios registrados.
func (s *UserService) ListUsers() ([]domain.User, error) {
	return s.store.ListUsers()
}

// isValidRole limita los roles posibles para mantener reglas claras del dominio.
func isValidRole(role domain.Role) bool {
	switch role {
	case domain.RoleAdmin, domain.RoleManager, domain.RoleCashier:
		return true
	default:
		return false
	}
}
