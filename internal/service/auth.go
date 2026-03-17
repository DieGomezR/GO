// Package service implementa los casos de uso y reglas del negocio.
package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"tienda-go/internal/domain"
	"tienda-go/internal/platform/ids"
	"tienda-go/internal/security"
	"tienda-go/internal/store"
)

// AuthService maneja login, validacion de tokens y cierre de sesiones.
type AuthService struct {
	store    store.Store
	tokenTTL time.Duration
}

// LoginInput representa el payload minimo para iniciar sesion.
type LoginInput struct {
	Email    string
	Password string
}

// AuthResult devuelve la sesion creada junto con el usuario autenticado.
type AuthResult struct {
	User      domain.User
	Token     string
	ExpiresAt time.Time
}

// NewAuthService construye el servicio de autenticacion.
func NewAuthService(store store.Store, tokenTTL time.Duration) *AuthService {
	return &AuthService{
		store:    store,
		tokenTTL: tokenTTL,
	}
}

// Login valida credenciales y crea una sesion nueva.
func (s *AuthService) Login(input LoginInput) (AuthResult, error) {
	if strings.TrimSpace(input.Email) == "" || strings.TrimSpace(input.Password) == "" {
		return AuthResult{}, fmt.Errorf("%w: email and password are required", domain.ErrValidation)
	}

	user, err := s.store.GetUserByEmail(strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return AuthResult{}, domain.ErrInvalidCredentials
		}

		return AuthResult{}, err
	}

	if !user.Active || !security.ComparePassword(input.Password, user.PasswordSalt, user.PasswordHash) {
		return AuthResult{}, domain.ErrInvalidCredentials
	}

	now := time.Now().UTC()
	// El token es opaco y se resuelve contra el store en cada request autenticado.
	session := domain.Session{
		Token:     ids.New("tok"),
		UserID:    user.ID,
		ExpiresAt: now.Add(s.tokenTTL),
		CreatedAt: now,
	}
	if err := s.store.CreateSession(session); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		User:      user,
		Token:     session.Token,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// Authenticate resuelve un token hasta el usuario asociado y verifica expiracion.
func (s *AuthService) Authenticate(token string) (domain.User, error) {
	if strings.TrimSpace(token) == "" {
		return domain.User{}, domain.ErrUnauthorized
	}

	session, err := s.store.GetSession(strings.TrimSpace(token))
	if err != nil {
		return domain.User{}, err
	}

	if time.Now().UTC().After(session.ExpiresAt) {
		// Si el token expiro, se limpia para no dejar basura en memoria.
		_ = s.store.DeleteSession(token)
		return domain.User{}, domain.ErrExpiredSession
	}

	user, err := s.store.GetUserByID(session.UserID)
	if err != nil {
		return domain.User{}, domain.ErrUnauthorized
	}

	if !user.Active {
		return domain.User{}, domain.ErrForbidden
	}

	return user, nil
}

// Logout elimina la sesion actual. Se deja idempotente para simplificar clientes.
func (s *AuthService) Logout(token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}

	return s.store.DeleteSession(strings.TrimSpace(token))
}
