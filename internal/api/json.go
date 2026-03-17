// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"tienda-go/internal/domain"
)

// writeJSON centraliza la escritura de respuestas JSON.
func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}

	_ = json.NewEncoder(w).Encode(payload)
}

// decodeJSON valida que el body tenga un solo objeto JSON y que no traiga campos desconocidos.
func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
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
func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{"data": data})
}

// writeError traduce errores de dominio a codigos HTTP estables para el cliente.
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError

	switch {
	case errors.Is(err, domain.ErrValidation):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrUnauthorized), errors.Is(err, domain.ErrExpiredSession), errors.Is(err, domain.ErrInvalidCredentials):
		status = http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		status = http.StatusForbidden
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInsufficientStock), errors.Is(err, domain.ErrInactiveProduct):
		status = http.StatusConflict
	}

	writeJSON(w, status, map[string]any{
		"error": err.Error(),
	})
}
