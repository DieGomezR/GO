// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"time"

	"tienda-go/internal/domain"
	"tienda-go/internal/service"
)

// userResponse define la forma publica de un usuario en la API.
type userResponse struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"created_at"`
}

// productResponse define la forma publica de un producto.
type productResponse struct {
	ID          string    `json:"id"`
	SKU         string    `json:"sku"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PriceCents  int64     `json:"price_cents"`
	Stock       int64     `json:"stock"`
	Active      bool      `json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// inventoryMovementResponse serializa un movimiento de inventario.
type inventoryMovementResponse struct {
	ID        string              `json:"id"`
	ProductID string              `json:"product_id"`
	Type      domain.MovementType `json:"type"`
	Quantity  int64               `json:"quantity"`
	Note      string              `json:"note"`
	CreatedBy string              `json:"created_by"`
	CreatedAt time.Time           `json:"created_at"`
}

// orderItemResponse serializa cada item dentro de una orden.
type orderItemResponse struct {
	ProductID      string `json:"product_id"`
	ProductName    string `json:"product_name"`
	Quantity       int64  `json:"quantity"`
	UnitPriceCents int64  `json:"unit_price_cents"`
	SubtotalCents  int64  `json:"subtotal_cents"`
}

// orderResponse serializa la venta completa.
type orderResponse struct {
	ID           string              `json:"id"`
	Number       string              `json:"number"`
	CustomerName string              `json:"customer_name"`
	CashierID    string              `json:"cashier_id"`
	Status       domain.OrderStatus  `json:"status"`
	Items        []orderItemResponse `json:"items"`
	TotalCents   int64               `json:"total_cents"`
	CreatedAt    time.Time           `json:"created_at"`
}

// summaryResponse representa el dashboard ya adaptado a JSON.
type summaryResponse struct {
	TotalUsers       int               `json:"total_users"`
	TotalProducts    int               `json:"total_products"`
	TotalOrders      int               `json:"total_orders"`
	RevenueCents     int64             `json:"revenue_cents"`
	LowStockProducts []productResponse `json:"low_stock_products"`
}

// presentUser convierte la entidad interna a un DTO HTTP sin exponer hashes ni salts.
func presentUser(user domain.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Role:      user.Role,
		Active:    user.Active,
		CreatedAt: user.CreatedAt,
	}
}

// presentUsers mapea una lista de usuarios.
func presentUsers(users []domain.User) []userResponse {
	response := make([]userResponse, 0, len(users))
	for _, user := range users {
		response = append(response, presentUser(user))
	}

	return response
}

// presentProduct convierte la entidad Product al formato HTTP.
func presentProduct(product domain.Product) productResponse {
	return productResponse{
		ID:          product.ID,
		SKU:         product.SKU,
		Name:        product.Name,
		Description: product.Description,
		PriceCents:  product.PriceCents,
		Stock:       product.Stock,
		Active:      product.Active,
		CreatedAt:   product.CreatedAt,
		UpdatedAt:   product.UpdatedAt,
	}
}

// presentProducts mapea una lista de productos.
func presentProducts(products []domain.Product) []productResponse {
	response := make([]productResponse, 0, len(products))
	for _, product := range products {
		response = append(response, presentProduct(product))
	}

	return response
}

// presentInventory mapea el historial de inventario.
func presentInventory(movements []domain.InventoryMovement) []inventoryMovementResponse {
	response := make([]inventoryMovementResponse, 0, len(movements))
	for _, movement := range movements {
		response = append(response, inventoryMovementResponse{
			ID:        movement.ID,
			ProductID: movement.ProductID,
			Type:      movement.Type,
			Quantity:  movement.Quantity,
			Note:      movement.Note,
			CreatedBy: movement.CreatedBy,
			CreatedAt: movement.CreatedAt,
		})
	}

	return response
}

// presentOrder adapta la orden y sus items a la forma de salida JSON.
func presentOrder(order domain.Order) orderResponse {
	items := make([]orderItemResponse, 0, len(order.Items))
	for _, item := range order.Items {
		items = append(items, orderItemResponse{
			ProductID:      item.ProductID,
			ProductName:    item.ProductName,
			Quantity:       item.Quantity,
			UnitPriceCents: item.UnitPriceCents,
			SubtotalCents:  item.SubtotalCents,
		})
	}

	return orderResponse{
		ID:           order.ID,
		Number:       order.Number,
		CustomerName: order.CustomerName,
		CashierID:    order.CashierID,
		Status:       order.Status,
		Items:        items,
		TotalCents:   order.TotalCents,
		CreatedAt:    order.CreatedAt,
	}
}

// presentOrders mapea varias ordenes.
func presentOrders(orders []domain.Order) []orderResponse {
	response := make([]orderResponse, 0, len(orders))
	for _, order := range orders {
		response = append(response, presentOrder(order))
	}

	return response
}

// presentSummary adapta el resumen agregado del dashboard.
func presentSummary(summary service.Summary) summaryResponse {
	return summaryResponse{
		TotalUsers:       summary.TotalUsers,
		TotalProducts:    summary.TotalProducts,
		TotalOrders:      summary.TotalOrders,
		RevenueCents:     summary.RevenueCents,
		LowStockProducts: presentProducts(summary.LowStockProducts),
	}
}
