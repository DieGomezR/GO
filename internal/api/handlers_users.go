// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/domain"
	"tienda-go/internal/service"
)

// createUserRequest define el payload esperado para registrar usuarios.
type createUserRequest struct {
	Name     string      `json:"name"`
	Email    string      `json:"email"`
	Password string      `json:"password"`
	Role     domain.Role `json:"role"`
}

// handleCreateUser crea una cuenta usando el servicio de usuarios.
func (a *API) handleCreateUser(c *fiber.Ctx) error {
	var request createUserRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}

	user, err := a.users.CreateUser(service.CreateUserInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
		Role:     request.Role,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusCreated, presentUser(user))
}

// handleListUsers devuelve todos los usuarios visibles para roles administrativos.
func (a *API) handleListUsers(c *fiber.Ctx) error {
	users, err := a.users.ListUsers()
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentUsers(users))
}

// handleDashboardSummary expone el resumen agregado de la tienda.
func (a *API) handleDashboardSummary(c *fiber.Ctx) error {
	summary, err := a.dashboard.Summary(a.cfg.LowStockLimit)
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentSummary(summary))
}
