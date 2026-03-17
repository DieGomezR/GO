// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/domain"
)

// writeJSON centraliza la escritura de respuestas JSON.
func writeJSON(c *fiber.Ctx, status int, payload any) error {
	c.Status(status)
	if payload == nil {
		return nil
	}

	return c.JSON(payload)
}

// decodeJSON valida que el body tenga un solo objeto JSON y que no traiga campos desconocidos.
func decodeJSON(c *fiber.Ctx, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrValidation, err)
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: body contains multiple JSON values", domain.ErrValidation)
	}

	return nil
}

// writeData envuelve las respuestas exitosas en una clave "data" consistente.
func writeData(c *fiber.Ctx, status int, data any) error {
	return writeJSON(c, status, map[string]any{"data": data})
}

// writeError traduce errores de dominio a codigos HTTP estables para el cliente.
func writeError(c *fiber.Ctx, err error) error {
	status := fiber.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrValidation):
		status = fiber.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrExpiredSession), errors.Is(err, domain.ErrInvalidCredentials):
		status = fiber.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status = fiber.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		status = fiber.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInsufficientStock), errors.Is(err, domain.ErrInactiveProduct):
		status = fiber.StatusConflict
	}

	return writeJSON(c, status, map[string]any{
		"error": err.Error(),
	})
}
