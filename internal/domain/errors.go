// Package domain define el lenguaje del negocio: entidades, tipos y errores
// comunes que luego reutilizan servicios, storage y API.
package domain

import "errors"

var (
	// ErrNotFound indica que un recurso no existe o no fue encontrado.
	ErrNotFound = errors.New("resource not found")
	// ErrUnauthorized indica que falta autenticacion valida.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden indica que el usuario esta autenticado pero no tiene permisos.
	ErrForbidden = errors.New("forbidden")
	// ErrValidation se usa para inputs invalidos o incompletos.
	ErrValidation = errors.New("validation error")
	// ErrConflict representa duplicados o estados incompatibles.
	ErrConflict = errors.New("conflict")
	// ErrInsufficientStock impide ventas o ajustes que dejen stock negativo.
	ErrInsufficientStock = errors.New("insufficient stock")
	// ErrInactiveProduct bloquea la venta de productos deshabilitados.
	ErrInactiveProduct = errors.New("product is inactive")
	// ErrInvalidCredentials cubre combinaciones email/password incorrectas.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrExpiredSession se devuelve cuando un token ya no es valido por tiempo.
	ErrExpiredSession = errors.New("expired session")
)
