// Package domain define el lenguaje del negocio: entidades, tipos y errores
// comunes que luego reutilizan servicios, storage y API.
package domain

import "time"

// OrderStatus representa el estado comercial de una orden.
type OrderStatus string

const (
	// OrderStatusPaid simplifica el ejemplo asumiendo ventas pagadas al crearse.
	OrderStatusPaid OrderStatus = "paid"
)

// OrderItem captura el snapshot del producto vendido en el momento de la venta.
type OrderItem struct {
	ProductID      string
	ProductName    string
	Quantity       int64
	UnitPriceCents int64
	SubtotalCents  int64
}

// Order representa la venta ya confirmada.
type Order struct {
	ID           string
	Number       string
	CustomerName string
	CashierID    string
	Status       OrderStatus
	Items        []OrderItem
	TotalCents   int64
	CreatedAt    time.Time
}
