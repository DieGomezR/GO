// Package store define los contratos de persistencia compartidos por la
// implementacion en memoria y la implementacion basada en MySQL.
package store

import "tienda-go/internal/domain"

// Store expone todas las operaciones de persistencia que necesita la tienda.
// En este proyecto se mantiene como una interfaz unica para no introducir
// demasiadas capas mientras aprendemos la separacion entre negocio y storage.
type Store interface {
	CreateUser(user domain.User) error
	GetUserByEmail(email string) (domain.User, error)
	GetUserByID(id string) (domain.User, error)
	ListUsers() ([]domain.User, error)

	CreateSession(session domain.Session) error
	GetSession(token string) (domain.Session, error)
	DeleteSession(token string) error

	CreateProduct(product domain.Product) error
	UpdateProduct(product domain.Product) error
	GetProduct(id string) (domain.Product, error)
	ListProducts() ([]domain.Product, error)

	AddMovement(movement domain.InventoryMovement) error
	ListProductMovements(productID string) ([]domain.InventoryMovement, error)

	CreateOrder(order domain.Order) error
	GetOrder(id string) (domain.Order, error)
	ListOrders() ([]domain.Order, error)

	// WithLock serializa operaciones de varios pasos dentro del mismo proceso.
	// En memoria evita inconsistencias por intercalado, y con MySQL mantiene el
	// ejemplo sencillo sin introducir transacciones complejas en los servicios.
	WithLock(fn func() error) error
}
