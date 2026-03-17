// Package service prueba los casos de uso mas importantes del proyecto.
package service

import (
	"errors"
	"testing"

	"tienda-go/internal/domain"
	"tienda-go/internal/store"
)

// TestOrderServiceCreateOrderDiscountsStock valida el camino feliz de una venta.
func TestOrderServiceCreateOrderDiscountsStock(t *testing.T) {
	t.Parallel()

	memoryStore := store.NewMemoryStore()
	userService := NewUserService(memoryStore)
	productService := NewProductService(memoryStore)
	inventoryService := NewInventoryService(memoryStore)
	orderService := NewOrderService(memoryStore)

	cashier, err := userService.CreateUser(CreateUserInput{
		Name:     "Caja",
		Email:    "cashier@example.com",
		Password: "Cashier1234!",
		Role:     domain.RoleCashier,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	product, err := productService.CreateProduct(CreateProductInput{
		SKU:          "PRD-01",
		Name:         "Producto Demo",
		Description:  "Producto para pruebas",
		PriceCents:   1500,
		InitialStock: 10,
		ActorID:      cashier.ID,
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	order, err := orderService.CreateOrder(CreateOrderInput{
		CustomerName: "Cliente Demo",
		CashierID:    cashier.ID,
		Items: []CreateOrderItemInput{
			{ProductID: product.ID, Quantity: 3},
		},
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	if order.TotalCents != 4500 {
		t.Fatalf("expected total 4500, got %d", order.TotalCents)
	}

	updatedProduct, err := productService.GetProduct(product.ID)
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}

	if updatedProduct.Stock != 7 {
		t.Fatalf("expected stock 7, got %d", updatedProduct.Stock)
	}

	// El producto nace con un movimiento inicial de restock y luego debe aparecer la venta.
	movements, err := inventoryService.ListMovements(product.ID)
	if err != nil {
		t.Fatalf("ListMovements() error = %v", err)
	}

	if len(movements) != 2 {
		t.Fatalf("expected 2 movements, got %d", len(movements))
	}

	if movements[0].Type != domain.MovementSale || movements[0].Quantity != -3 {
		t.Fatalf("expected latest movement to be sale -3, got %s %d", movements[0].Type, movements[0].Quantity)
	}
}

// TestOrderServiceCreateOrderFailsWhenStockIsInsufficient asegura que no se muta el stock al fallar.
func TestOrderServiceCreateOrderFailsWhenStockIsInsufficient(t *testing.T) {
	t.Parallel()

	memoryStore := store.NewMemoryStore()
	userService := NewUserService(memoryStore)
	productService := NewProductService(memoryStore)
	orderService := NewOrderService(memoryStore)

	cashier, err := userService.CreateUser(CreateUserInput{
		Name:     "Caja",
		Email:    "cashier2@example.com",
		Password: "Cashier1234!",
		Role:     domain.RoleCashier,
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	product, err := productService.CreateProduct(CreateProductInput{
		SKU:          "PRD-02",
		Name:         "Producto Limitado",
		Description:  "Producto con poco inventario",
		PriceCents:   2000,
		InitialStock: 2,
		ActorID:      cashier.ID,
	})
	if err != nil {
		t.Fatalf("CreateProduct() error = %v", err)
	}

	_, err = orderService.CreateOrder(CreateOrderInput{
		CustomerName: "Cliente Demo",
		CashierID:    cashier.ID,
		Items: []CreateOrderItemInput{
			{ProductID: product.ID, Quantity: 4},
		},
	})
	if !errors.Is(err, domain.ErrInsufficientStock) {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}

	updatedProduct, err := productService.GetProduct(product.ID)
	if err != nil {
		t.Fatalf("GetProduct() error = %v", err)
	}

	if updatedProduct.Stock != 2 {
		t.Fatalf("expected stock to remain 2, got %d", updatedProduct.Stock)
	}
}
