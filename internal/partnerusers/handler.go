package partnerusers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// CurrentUserFunc desacopla el handler de cualquier middleware concreto.
type CurrentUserFunc func(r *http.Request) AuthUser

// Handler expone por HTTP los casos de uso que nacieron del controller PHP.
type Handler struct {
	service     *Service
	currentUser CurrentUserFunc
}

// NewHandler construye el adaptador HTTP.
func NewHandler(service *Service, currentUser CurrentUserFunc) *Handler {
	if currentUser == nil {
		currentUser = func(_ *http.Request) AuthUser { return AuthUser{} }
	}
	return &Handler{service: service, currentUser: currentUser}
}

// RegisterRoutes muestra como quedarian distribuidas las rutas del controller legacy.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/partner-users/register", h.handleRegister)
	mux.HandleFunc("PUT /v1/partner-users/update", h.handleUpdate)
	mux.HandleFunc("POST /v1/partner-users/activate", h.handleActivatePackage)
	mux.HandleFunc("DELETE /v1/partner-users/remove", h.handleRemove)
	mux.HandleFunc("POST /v1/partner-users/activate-channels", h.handleActivateChannels)
	mux.HandleFunc("GET /v1/partner-users/details", h.handleGetUserDetails)
	mux.HandleFunc("GET /v1/partner-users/by-partner", h.handleListUsersByPartner)
	mux.HandleFunc("GET /v1/partner-users/early-deactivation-status", h.handleEarlyDeactivationStatus)
	mux.HandleFunc("POST /v1/partner-users/deactivate", h.handleDeactivate)
	mux.HandleFunc("POST /v1/partner-users/upload", h.handleUpload)
	mux.HandleFunc("POST /v1/partner-users/bulk-users", h.handleUpload)
	mux.HandleFunc("POST /v1/partner-users/deactivate-multiple", h.handleDeactivateMultiple)
	mux.HandleFunc("POST /v1/partner-users/change-package-multiple", h.handleChangePackageMultiple)
	mux.HandleFunc("DELETE /v1/partner-users/remove-multiple", h.handleRemoveMultiple)
	mux.HandleFunc("POST /v1/partner-users/generate-login-token", h.handleGenerateLoginToken)
	mux.HandleFunc("GET /v1/partner-users/generate-login-token-by-email", h.handleGenerateLoginTokenByEmail)
	mux.HandleFunc("POST /v1/partner-users/logout-all-devices", h.handleLogoutAllDevices)
	mux.HandleFunc("POST /v1/partner-users/reactivate", h.handleReactivate)
}

// RegisterProdOnlyRoutes registra rutas que solo dependen de db_prod.
func (h *Handler) RegisterProdOnlyRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/partner-users/prod/by-partner", h.handleListUsersByPartnerFromProd)
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.Register(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.Update(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleActivatePackage(w http.ResponseWriter, r *http.Request) {
	var req ActivatePackageRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.ActivatePackage(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleRemove(w http.ResponseWriter, r *http.Request) {
	var req RemoveRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.Remove(r.Context(), req, h.currentUser(r), clientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleActivateChannels(w http.ResponseWriter, r *http.Request) {
	var req ActivateChannelsRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.ActivateChannels(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGetUserDetails(w http.ResponseWriter, r *http.Request) {
	req := UserDetailsRequest{PartnerID: r.URL.Query().Get("partnerId")}
	result, err := h.service.GetUserDetails(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleListUsersByPartner(w http.ResponseWriter, r *http.Request) {
	req := ListUsersByPartnerRequest{
		Partner: r.URL.Query().Get("partner"),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := parseInt64(rawLimit)
		if err != nil {
			writeError(w, err)
			return
		}
		req.Limit = int(limit)
	}

	result, err := h.service.ListUsersByPartner(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleListUsersByPartnerFromProd(w http.ResponseWriter, r *http.Request) {
	req := ListUsersByPartnerRequest{
		Partner: r.URL.Query().Get("partner"),
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := parseInt64(rawLimit)
		if err != nil {
			writeError(w, err)
			return
		}
		req.Limit = int(limit)
	}

	result, err := h.service.ListUsersByPartnerFromProd(r.Context(), req)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleEarlyDeactivationStatus(w http.ResponseWriter, r *http.Request) {
	req := EarlyDeactivationStatusRequest{Country: r.URL.Query().Get("country")}
	result, err := h.service.GetEarlyDeactivationStatus(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleDeactivate(w http.ResponseWriter, r *http.Request) {
	var req DeactivateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.Deactivate(r.Context(), req, h.currentUser(r), clientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleUpload(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("archivo")
	if err != nil {
		writeError(w, fmt.Errorf("%w: archivo is required", ErrValidation))
		return
	}
	defer file.Close()

	result, err := h.service.QueueBulkUpload(r.Context(), QueueBulkUploadRequest{
		FileName: header.Filename,
		Content:  file,
	}, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, result)
}

func (h *Handler) handleDeactivateMultiple(w http.ResponseWriter, r *http.Request) {
	var req BulkOperationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.DeactivateMultiple(r.Context(), req, h.currentUser(r), clientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Proceso de desactivacion masiva completado.",
		"results": result,
	})
}

func (h *Handler) handleChangePackageMultiple(w http.ResponseWriter, r *http.Request) {
	var req ChangePackageMultipleRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.ChangePackageMultiple(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Cambio de paquete masivo completado.",
		"results": result,
	})
}

func (h *Handler) handleRemoveMultiple(w http.ResponseWriter, r *http.Request) {
	var req BulkOperationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.RemoveMultiple(r.Context(), req, h.currentUser(r), clientIP(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "Proceso de eliminacion masiva completado.",
		"results": result,
	})
}

func (h *Handler) handleGenerateLoginToken(w http.ResponseWriter, r *http.Request) {
	var req GenerateLoginTokenRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.GenerateUserLoginToken(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleGenerateLoginTokenByEmail(w http.ResponseWriter, r *http.Request) {
	req := GenerateLoginTokenByEmailRequest{Email: r.URL.Query().Get("email")}
	result, err := h.service.GenerateUserLoginTokenByEmail(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleLogoutAllDevices(w http.ResponseWriter, r *http.Request) {
	var req LogoutAllDevicesRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.LogoutAllDevices(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) handleReactivate(w http.ResponseWriter, r *http.Request) {
	var req ReactivateDeletedUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, err)
		return
	}
	result, err := h.service.ReactivateDeletedUser(r.Context(), req, h.currentUser(r))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: body contains multiple JSON values", ErrValidation)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		status = http.StatusForbidden
	case errors.Is(err, ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrConflict):
		status = http.StatusConflict
	}

	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	return strings.TrimSpace(r.RemoteAddr)
}
