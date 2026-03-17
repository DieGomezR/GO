package partnerusers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// CurrentUserFunc desacopla el handler de cualquier middleware concreto.
type CurrentUserFunc func(c *fiber.Ctx) AuthUser

// Handler expone por HTTP los casos de uso que nacieron del controller PHP.
type Handler struct {
	service     *Service
	currentUser CurrentUserFunc
}

// NewHandler construye el adaptador HTTP.
func NewHandler(service *Service, currentUser CurrentUserFunc) *Handler {
	if currentUser == nil {
		currentUser = func(_ *fiber.Ctx) AuthUser { return AuthUser{} }
	}
	return &Handler{service: service, currentUser: currentUser}
}

// RegisterRoutes muestra como quedarian distribuidas las rutas del controller legacy.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Post("/register", h.handleRegister)
	router.Put("/update", h.handleUpdate)
	router.Post("/activate", h.handleActivatePackage)
	router.Delete("/remove", h.handleRemove)
	router.Post("/activate-channels", h.handleActivateChannels)
	router.Get("/details", h.handleGetUserDetails)
	router.Get("/by-partner", h.handleListUsersByPartner)
	router.Get("/early-deactivation-status", h.handleEarlyDeactivationStatus)
	router.Post("/deactivate", h.handleDeactivate)
	router.Post("/upload", h.handleUpload)
	router.Post("/bulk-users", h.handleUpload)
	router.Post("/deactivate-multiple", h.handleDeactivateMultiple)
	router.Post("/change-package-multiple", h.handleChangePackageMultiple)
	router.Delete("/remove-multiple", h.handleRemoveMultiple)
	router.Post("/generate-login-token", h.handleGenerateLoginToken)
	router.Get("/generate-login-token-by-email", h.handleGenerateLoginTokenByEmail)
	router.Post("/logout-all-devices", h.handleLogoutAllDevices)
	router.Post("/reactivate", h.handleReactivate)
}

// RegisterProdOnlyRoutes registra rutas que solo dependen de db_prod.
func (h *Handler) RegisterProdOnlyRoutes(router fiber.Router) {
	router.Get("/prod/by-partner", h.handleListUsersByPartnerFromProd)
}

func (h *Handler) handleRegister(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.Register(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, result)
}

func (h *Handler) handleUpdate(c *fiber.Ctx) error {
	var req UpdateRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.Update(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleActivatePackage(c *fiber.Ctx) error {
	var req ActivatePackageRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.ActivatePackage(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleRemove(c *fiber.Ctx) error {
	var req RemoveRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.Remove(c.UserContext(), req, h.currentUser(c), clientIP(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleActivateChannels(c *fiber.Ctx) error {
	var req ActivateChannelsRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.ActivateChannels(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleGetUserDetails(c *fiber.Ctx) error {
	req := UserDetailsRequest{PartnerID: c.Query("partnerId")}
	result, err := h.service.GetUserDetails(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleListUsersByPartner(c *fiber.Ctx) error {
	req := ListUsersByPartnerRequest{Partner: c.Query("partner")}
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		limit, err := parseInt64(rawLimit)
		if err != nil {
			return writeError(c, err)
		}
		req.Limit = int(limit)
	}

	result, err := h.service.ListUsersByPartner(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleListUsersByPartnerFromProd(c *fiber.Ctx) error {
	req := ListUsersByPartnerRequest{Partner: c.Query("partner")}
	if rawLimit := strings.TrimSpace(c.Query("limit")); rawLimit != "" {
		limit, err := parseInt64(rawLimit)
		if err != nil {
			return writeError(c, err)
		}
		req.Limit = int(limit)
	}

	result, err := h.service.ListUsersByPartnerFromProd(c.UserContext(), req)
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleEarlyDeactivationStatus(c *fiber.Ctx) error {
	req := EarlyDeactivationStatusRequest{Country: c.Query("country")}
	result, err := h.service.GetEarlyDeactivationStatus(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleDeactivate(c *fiber.Ctx) error {
	var req DeactivateRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.Deactivate(c.UserContext(), req, h.currentUser(c), clientIP(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleUpload(c *fiber.Ctx) error {
	header, err := c.FormFile("archivo")
	if err != nil {
		return writeError(c, fmt.Errorf("%w: archivo is required", ErrValidation))
	}

	file, err := header.Open()
	if err != nil {
		return writeError(c, err)
	}
	defer file.Close()

	result, err := h.service.QueueBulkUpload(c.UserContext(), QueueBulkUploadRequest{
		FileName: header.Filename,
		Content:  file,
	}, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusAccepted, result)
}

func (h *Handler) handleDeactivateMultiple(c *fiber.Ctx) error {
	var req BulkOperationRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.DeactivateMultiple(c.UserContext(), req, h.currentUser(c), clientIP(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"message": "Proceso de desactivacion masiva completado.",
		"results": result,
	})
}

func (h *Handler) handleChangePackageMultiple(c *fiber.Ctx) error {
	var req ChangePackageMultipleRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.ChangePackageMultiple(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"message": "Cambio de paquete masivo completado.",
		"results": result,
	})
}

func (h *Handler) handleRemoveMultiple(c *fiber.Ctx) error {
	var req BulkOperationRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.RemoveMultiple(c.UserContext(), req, h.currentUser(c), clientIP(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, map[string]any{
		"message": "Proceso de eliminacion masiva completado.",
		"results": result,
	})
}

func (h *Handler) handleGenerateLoginToken(c *fiber.Ctx) error {
	var req GenerateLoginTokenRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.GenerateUserLoginToken(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleGenerateLoginTokenByEmail(c *fiber.Ctx) error {
	req := GenerateLoginTokenByEmailRequest{Email: c.Query("email")}
	result, err := h.service.GenerateUserLoginTokenByEmail(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleLogoutAllDevices(c *fiber.Ctx) error {
	var req LogoutAllDevicesRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.LogoutAllDevices(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusOK, result)
}

func (h *Handler) handleReactivate(c *fiber.Ctx) error {
	var req ReactivateDeletedUserRequest
	if err := decodeJSON(c, &req); err != nil {
		return writeError(c, err)
	}
	result, err := h.service.ReactivateDeletedUser(c.UserContext(), req, h.currentUser(c))
	if err != nil {
		return writeError(c, err)
	}
	return writeJSON(c, fiber.StatusCreated, result)
}

func decodeJSON(c *fiber.Ctx, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", ErrValidation, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: body contains multiple JSON values", ErrValidation)
	}
	return nil
}

func writeJSON(c *fiber.Ctx, status int, payload any) error {
	c.Status(status)
	if payload == nil {
		return nil
	}
	return c.JSON(payload)
}

func writeError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError
	switch {
	case errors.Is(err, ErrValidation):
		status = fiber.StatusBadRequest
	case errors.Is(err, ErrUnauthorized):
		status = fiber.StatusForbidden
	case errors.Is(err, ErrForbidden):
		status = fiber.StatusForbidden
	case errors.Is(err, ErrNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, ErrConflict):
		status = fiber.StatusConflict
	}

	return writeJSON(c, status, map[string]any{"error": err.Error()})
}

func clientIP(c *fiber.Ctx) string {
	return strings.TrimSpace(c.IP())
}
