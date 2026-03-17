// Package service implementa los casos de uso y reglas del negocio.
package service

import (
	"tienda-go/internal/domain"
	"tienda-go/internal/store"
)

// DashboardService calcula vistas agregadas para administracion.
type DashboardService struct {
	store store.Store
}

// Summary resume el estado actual del negocio.
type Summary struct {
	TotalUsers       int              `json:"total_users"`
	TotalProducts    int              `json:"total_products"`
	TotalOrders      int              `json:"total_orders"`
	RevenueCents     int64            `json:"revenue_cents"`
	LowStockProducts []domain.Product `json:"low_stock_products"`
}

// NewDashboardService construye el servicio de resumen.
func NewDashboardService(store store.Store) *DashboardService {
	return &DashboardService{store: store}
}

// Summary calcula metricas rapidas sin cambiar el estado del sistema.
func (s *DashboardService) Summary(lowStockLimit int64) (Summary, error) {
	users, err := s.store.ListUsers()
	if err != nil {
		return Summary{}, err
	}

	products, err := s.store.ListProducts()
	if err != nil {
		return Summary{}, err
	}

	orders, err := s.store.ListOrders()
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{
		TotalUsers:    len(users),
		TotalProducts: len(products),
		TotalOrders:   len(orders),
	}

	for _, order := range orders {
		summary.RevenueCents += order.TotalCents
	}

	// Se consideran criticos los productos activos con stock en o por debajo del limite.
	for _, product := range products {
		if product.Active && product.Stock <= lowStockLimit {
			summary.LowStockProducts = append(summary.LowStockProducts, product)
		}
	}

	return summary, nil
}
