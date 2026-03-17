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

// OrderService gestiona la venta y protege la seccion critica de descuento de stock.
type OrderService struct {
	store store.Store
}

// CreateOrderItemInput representa cada item solicitado por el cliente.
type CreateOrderItemInput struct {
	ProductID string
	Quantity  int64
}

// CreateOrderInput describe los datos necesarios para registrar una venta.
type CreateOrderInput struct {
	CustomerName string
	CashierID    string
	Items        []CreateOrderItemInput
}

// NewOrderService construye el servicio de ordenes.
func NewOrderService(store store.Store) *OrderService {
	return &OrderService{store: store}
}

// CreateOrder valida los items, verifica stock, descuenta inventario y persiste la orden.
func (s *OrderService) CreateOrder(input CreateOrderInput) (domain.Order, error) {
	if len(input.Items) == 0 {
		return domain.Order{}, fmt.Errorf("%w: at least one order item is required", domain.ErrValidation)
	}

	if _, err := s.store.GetUserByID(strings.TrimSpace(input.CashierID)); err != nil {
		return domain.Order{}, fmt.Errorf("%w: cashier not found", domain.ErrValidation)
	}

	itemsByProduct := make(map[string]int64)
	for _, item := range input.Items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" || item.Quantity <= 0 {
			return domain.Order{}, fmt.Errorf("%w: each item requires a product id and a quantity greater than zero", domain.ErrValidation)
		}

		itemsByProduct[productID] += item.Quantity
	}

	var order domain.Order
	err := s.store.WithLock(func() error {
		now := time.Now().UTC()
		orderNumber := ids.OrderNumber()
		orderItems := make([]domain.OrderItem, 0, len(itemsByProduct))
		totalCents := int64(0)
		products := make(map[string]domain.Product, len(itemsByProduct))

		// Primer recorrido: validar todo antes de modificar el estado.
		for productID, quantity := range itemsByProduct {
			product, err := s.store.GetProduct(productID)
			if err != nil {
				return err
			}

			if !product.Active {
				return domain.ErrInactiveProduct
			}

			if product.Stock < quantity {
				return domain.ErrInsufficientStock
			}

			products[productID] = product
		}

		// Segundo recorrido: como ya validamos todo, ahora aplicamos cambios.
		for productID, quantity := range itemsByProduct {
			product := products[productID]
			product.Stock -= quantity
			product.UpdatedAt = now
			if err := s.store.UpdateProduct(product); err != nil {
				return err
			}

			subtotal := product.PriceCents * quantity
			totalCents += subtotal
			orderItems = append(orderItems, domain.OrderItem{
				ProductID:      product.ID,
				ProductName:    product.Name,
				Quantity:       quantity,
				UnitPriceCents: product.PriceCents,
				SubtotalCents:  subtotal,
			})

			if err := s.store.AddMovement(domain.InventoryMovement{
				ID:        ids.New("mov"),
				ProductID: product.ID,
				Type:      domain.MovementSale,
				Quantity:  -quantity,
				Note:      "sale " + orderNumber,
				CreatedBy: strings.TrimSpace(input.CashierID),
				CreatedAt: now,
			}); err != nil {
				return err
			}
		}

		order = domain.Order{
			ID:           ids.New("ord"),
			Number:       orderNumber,
			CustomerName: strings.TrimSpace(input.CustomerName),
			CashierID:    strings.TrimSpace(input.CashierID),
			Status:       domain.OrderStatusPaid,
			Items:        orderItems,
			TotalCents:   totalCents,
			CreatedAt:    now,
		}

		return s.store.CreateOrder(order)
	})
	if err != nil {
		return domain.Order{}, err
	}

	return order, nil
}

// GetOrder devuelve una orden por ID.
func (s *OrderService) GetOrder(id string) (domain.Order, error) {
	return s.store.GetOrder(strings.TrimSpace(id))
}

// ListOrders devuelve el historial de ventas.
func (s *OrderService) ListOrders() ([]domain.Order, error) {
	return s.store.ListOrders()
}
