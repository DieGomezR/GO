// Package service prueba los casos de uso mas importantes del proyecto.
package service

import (
	"testing"
	"time"

	"tienda-go/internal/domain"
	"tienda-go/internal/store"
)

// TestAuthServiceLoginAndAuthenticate comprueba el flujo basico de login y resolucion de token.
func TestAuthServiceLoginAndAuthenticate(t *testing.T) {
	t.Parallel()

	memoryStore := store.NewMemoryStore()
	userService := NewUserService(memoryStore)
	authService := NewAuthService(memoryStore, 30*time.Minute)

	user, err := userService.CreateUser(CreateUserInput{
		Name:     "Admin Demo",
		Email:    "admin@example.com",
		Password: "Admin1234!",
		Role:     domain.RoleAdmin,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	result, err := authService.Login(LoginInput{
		Email:    user.Email,
		Password: "Admin1234!",
	})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if result.Token == "" {
		t.Fatal("expected token to be generated")
	}

	authenticatedUser, err := authService.Authenticate(result.Token)
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if authenticatedUser.ID != user.ID {
		t.Fatalf("expected authenticated user %q, got %q", user.ID, authenticatedUser.ID)
	}
}
