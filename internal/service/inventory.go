// Package service implementa los casos de uso y reglas del negocio.
package service

import (
	"fmt"
	"strings"
	"time"

	"tienda-go/internal/domain"
	"tienda-go/internal/platform/ids"
	"tienda-go/internal/store"
)

// InventoryService encapsula los ajustes manuales y consultas de inventario.
type InventoryService struct {
	store store.Store
}

// AdjustStockInput describe un movimiento manual sobre el stock.
type AdjustStockInput struct {
	Quantity int64
	Type     domain.MovementType
	Note     string
	ActorID  string
}

// NewInventoryService construye el servicio de inventario.
func NewInventoryService(store store.Store) *InventoryService {
	return &InventoryService{store: store}
}

// AdjustStock aplica un cambio de stock y deja evidencia en el historial.
func (s *InventoryService) AdjustStock(productID string, input AdjustStockInput) (domain.Product, error) {
	if input.Quantity == 0 {
		return domain.Product{}, fmt.Errorf("%w: quantity must be different from zero", domain.ErrValidation)
	}

	if input.Type != domain.MovementRestock && input.Type != domain.MovementAdjustment {
		return domain.Product{}, fmt.Errorf("%w: unsupported movement type %q", domain.ErrValidation, input.Type)
	}

	var product domain.Product
	err := s.store.WithLock(func() error {
		var err error
		product, err = s.store.GetProduct(strings.TrimSpace(productID))
		if err != nil {
			return err
		}

		nextStock := product.Stock + input.Quantity
		if nextStock < 0 {
			return domain.ErrInsufficientStock
		}

		// Primero se actualiza el producto; luego se registra el movimiento asociado.
		now := time.Now().UTC()
		product.Stock = nextStock
		product.UpdatedAt = now
		if err := s.store.UpdateProduct(product); err != nil {
			return err
		}

		movement := domain.InventoryMovement{
			ID:        ids.New("mov"),
			ProductID: product.ID,
			Type:      input.Type,
			Quantity:  input.Quantity,
			Note:      strings.TrimSpace(input.Note),
			CreatedBy: strings.TrimSpace(input.ActorID),
			CreatedAt: now,
		}
		return s.store.AddMovement(movement)
	})
	if err != nil {
		return domain.Product{}, err
	}

	return product, nil
}

// ListMovements devuelve el historial de movimientos de un producto.
func (s *InventoryService) ListMovements(productID string) ([]domain.InventoryMovement, error) {
	return s.store.ListProductMovements(strings.TrimSpace(productID))
}
