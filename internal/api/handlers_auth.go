// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"net/http"

	"tienda-go/internal/service"
)

// loginRequest representa el body esperado para iniciar sesion.
type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// handleHealth expone un endpoint minimo para verificar que el servicio esta vivo.
func (a *API) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeData(w, http.StatusOK, map[string]any{
		"app": a.cfg.AppName,
		"ok":  true,
	})
}

// handleLogin autentica al usuario y devuelve token + perfil.
func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	result, err := a.auth.Login(service.LoginInput{
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, map[string]any{
		"token":      result.Token,
		"expires_at": result.ExpiresAt,
		"user":       presentUser(result.User),
	})
}

// handleLogout invalida el token enviado en Authorization.
func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	if err := a.auth.Logout(extractBearerToken(r.Header.Get("Authorization"))); err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, map[string]any{"message": "session closed"})
}

// handleMe devuelve el usuario resuelto por el middleware de autenticacion.
func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	writeData(w, http.StatusOK, presentUser(currentUser(r)))
}
