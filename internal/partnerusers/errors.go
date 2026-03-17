package partnerusers

import "errors"

var (
	// ErrUnauthorized representa un request sin usuario autenticado utilizable.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden representa un request autenticado sin permisos.
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound indica que no existe el recurso buscado.
	ErrNotFound = errors.New("not found")
	// ErrValidation indica que la entrada del usuario no cumple reglas del caso de uso.
	ErrValidation = errors.New("validation error")
	// ErrConflict representa un estado incompatible con la operacion.
	ErrConflict = errors.New("conflict")
)
