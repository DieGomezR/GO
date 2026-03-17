// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/service"
)

// loginRequest representa el body esperado para iniciar sesion.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleHealth expone un endpoint minimo para verificar que el servicio esta vivo.
func (a *API) handleHealth(c *fiber.Ctx) error {
	return writeData(c, fiber.StatusOK, map[string]any{
		"app": a.cfg.AppName,
		"ok":  true,
	})
}

// handleLogin autentica al usuario y devuelve token + perfil.
func (a *API) handleLogin(c *fiber.Ctx) error {
	var request loginRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}

	result, err := a.auth.Login(service.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, map[string]any{
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
		"user":       presentUser(result.User),
	})
}

// handleLogout invalida el token enviado en Authorization.
func (a *API) handleLogout(c *fiber.Ctx) error {
	if err := a.auth.Logout(extractBearerToken(c.Get("Authorization"))); err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, map[string]any{"message": "session closed"})
}

// handleMe devuelve el usuario resuelto por el middleware de autenticacion.
func (a *API) handleMe(c *fiber.Ctx) error {
	return writeData(c, fiber.StatusOK, presentUser(currentUser(c)))
}
