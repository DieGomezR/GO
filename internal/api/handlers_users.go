// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"net/http"

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
func (a *API) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var request createUserRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	user, err := a.users.CreateUser(service.CreateUserInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
		Role:     request.Role,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusCreated, presentUser(user))
}

// handleListUsers devuelve todos los usuarios visibles para roles administrativos.
func (a *API) handleListUsers(w http.ResponseWriter, _ *http.Request) {
	users, err := a.users.ListUsers()
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentUsers(users))
}

// handleDashboardSummary expone el resumen agregado de la tienda.
func (a *API) handleDashboardSummary(w http.ResponseWriter, _ *http.Request) {
	summary, err := a.dashboard.Summary(a.cfg.LowStockLimit)
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentSummary(summary))
}
