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

// ProductService contiene la logica del catalogo.
type ProductService struct {
	store store.Store
}

// CreateProductInput define el payload necesario para crear un producto.
type CreateProductInput struct {
	SKU          string
	Name         string
	Description  string
	PriceCents   int64
	InitialStock int64
	ActorID      string
}

// UpdateProductInput define los campos editables del catalogo.
type UpdateProductInput struct {
	SKU         string
	Name        string
	Description string
	PriceCents  int64
	Active      bool
}

// NewProductService construye el servicio de productos.
func NewProductService(store store.Store) *ProductService {
	return &ProductService{store: store}
}

// CreateProduct valida el producto, lo persiste y registra stock inicial si aplica.
func (s *ProductService) CreateProduct(input CreateProductInput) (domain.Product, error) {
	sku := strings.ToUpper(strings.TrimSpace(input.SKU))
	name := strings.TrimSpace(input.Name)

	if sku == "" || name == "" {
		return domain.Product{}, fmt.Errorf("%w: sku and name are required", domain.ErrValidation)
	}

	if input.PriceCents <= 0 {
		return domain.Product{}, fmt.Errorf("%w: price must be greater than zero", domain.ErrValidation)
	}

	if input.InitialStock < 0 {
		return domain.Product{}, fmt.Errorf("%w: initial stock cannot be negative", domain.ErrValidation)
	}

	now := time.Now().UTC()
	product := domain.Product{
		ID:          ids.New("prd"),
		SKU:         sku,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		PriceCents:  input.PriceCents,
		Stock:       input.InitialStock,
		Active:      true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.store.WithLock(func() error {
		if err := s.store.CreateProduct(product); err != nil {
			return err
		}

		if input.InitialStock <= 0 {
			return nil
		}

		// Todo stock inicial tambien queda trazado como movimiento de inventario.
		return s.store.AddMovement(domain.InventoryMovement{
			ID:        ids.New("mov"),
			ProductID: product.ID,
			Type:      domain.MovementRestock,
			Quantity:  input.InitialStock,
			Note:      "initial stock",
			CreatedBy: strings.TrimSpace(input.ActorID),
			CreatedAt: now,
		})
	}); err != nil {
		return domain.Product{}, err
	}

	return product, nil
}

// UpdateProduct edita los campos del producto sin tocar el stock historico.
func (s *ProductService) UpdateProduct(id string, input UpdateProductInput) (domain.Product, error) {
	id = strings.TrimSpace(id)
	product, err := s.store.GetProduct(id)
	if err != nil {
		return domain.Product{}, err
	}

	sku := strings.ToUpper(strings.TrimSpace(input.SKU))
	name := strings.TrimSpace(input.Name)
	if sku == "" || name == "" {
		return domain.Product{}, fmt.Errorf("%w: sku and name are required", domain.ErrValidation)
	}

	if input.PriceCents <= 0 {
		return domain.Product{}, fmt.Errorf("%w: price must be greater than zero", domain.ErrValidation)
	}

	product.SKU = sku
	product.Name = name
	product.Description = strings.TrimSpace(input.Description)
	product.PriceCents = input.PriceCents
	product.Active = input.Active
	product.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateProduct(product); err != nil {
		return domain.Product{}, err
	}

	return product, nil
}

// GetProduct devuelve un producto por ID.
func (s *ProductService) GetProduct(id string) (domain.Product, error) {
	return s.store.GetProduct(strings.TrimSpace(id))
}

// ListProducts devuelve el catalogo completo.
func (s *ProductService) ListProducts() ([]domain.Product, error) {
	return s.store.ListProducts()
}
