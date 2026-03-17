// Package domain define el lenguaje del negocio: entidades, tipos y errores
// comunes que luego reutilizan servicios, storage y API.
package domain

import "time"

// Product representa un articulo vendible del catalogo.
type Product struct {
	ID          string
	SKU         string
	Name        string
	Description string
	PriceCents  int64
	Stock       int64
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MovementType clasifica los movimientos que afectan el inventario.
type MovementType string

const (
	// MovementRestock registra entradas planificadas de stock.
	MovementRestock MovementType = "restock"
	// MovementAdjustment registra correcciones manuales positivas o negativas.
	MovementAdjustment MovementType = "adjustment"
	// MovementSale registra la salida de stock causada por una venta.
	MovementSale MovementType = "sale"
)

// InventoryMovement deja trazabilidad de cada cambio de inventario.
type InventoryMovement struct {
	ID        string
	ProductID string
	Type      MovementType
	Quantity  int64
	Note      string
	CreatedBy string
	CreatedAt time.Time
}
