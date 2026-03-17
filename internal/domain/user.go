// Package domain define el lenguaje del negocio: entidades, tipos y errores
// comunes que luego reutilizan servicios, storage y API.
package domain

import "time"

// Role expresa el nivel de acceso de un usuario dentro de la tienda.
type Role string

const (
	// RoleAdmin tiene acceso total sobre la aplicacion.
	RoleAdmin Role = "admin"
	// RoleManager administra catalogo e inventario, pero no usuarios admin.
	RoleManager Role = "manager"
	// RoleCashier puede operar ventas y consultar recursos necesarios.
	RoleCashier Role = "cashier"
)

// User representa una cuenta del sistema.
// PasswordSalt y PasswordHash viven aqui porque esta version no separa
// credenciales en otra entidad para mantener el ejemplo corto.
type User struct {
	ID           string
	Name         string
	Email        string
	Role         Role
	PasswordSalt string
	PasswordHash string
	Active       bool
	CreatedAt    time.Time
}

// Session modela una sesion simple basada en token almacenado en memoria.
type Session struct {
	Token     string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}
