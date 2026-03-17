// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/domain"
	"tienda-go/internal/service"
)

// createProductRequest representa el body para crear un producto.
type createProductRequest struct {
	SKU          string `json:"sku"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	PriceCents   int64  `json:"price_cents"`
	InitialStock int64  `json:"initial_stock"`
}

// updateProductRequest representa el body para editar el catalogo.
type updateProductRequest struct {
	SKU         string `json:"sku"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PriceCents  int64  `json:"price_cents"`
	Active      bool   `json:"active"`
}

// adjustInventoryRequest representa un ajuste manual de inventario.
type adjustInventoryRequest struct {
	Quantity int64               `json:"quantity"`
	Type     domain.MovementType `json:"type"`
	Note     string              `json:"note"`
}

// handleCreateProduct delega la alta de productos al servicio de catalogo.
func (a *API) handleCreateProduct(c *fiber.Ctx) error {
	var request createProductRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}

	product, err := a.products.CreateProduct(service.CreateProductInput{
		SKU:          request.SKU,
		Name:         request.Name,
		Description:  request.Description,
		PriceCents:   request.PriceCents,
		InitialStock: request.InitialStock,
		ActorID:      currentUser(c).ID,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusCreated, presentProduct(product))
}

// handleUpdateProduct edita atributos del producto sin tocar el historial.
func (a *API) handleUpdateProduct(c *fiber.Ctx) error {
	var request updateProductRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}

	product, err := a.products.UpdateProduct(c.Params("id"), service.UpdateProductInput{
		SKU:         request.SKU,
		Name:        request.Name,
		Description: request.Description,
		PriceCents:  request.PriceCents,
		Active:      request.Active,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentProduct(product))
}

// handleListProducts devuelve el catalogo completo.
func (a *API) handleListProducts(c *fiber.Ctx) error {
	products, err := a.products.ListProducts()
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentProducts(products))
}

// handleGetProduct devuelve un producto puntual.
func (a *API) handleGetProduct(c *fiber.Ctx) error {
	product, err := a.products.GetProduct(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentProduct(product))
}

// handleAdjustInventory registra entradas o correcciones de stock.
func (a *API) handleAdjustInventory(c *fiber.Ctx) error {
	var request adjustInventoryRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
	}

	product, err := a.inventory.AdjustStock(c.Params("id"), service.AdjustStockInput{
		Quantity: request.Quantity,
		Type:     request.Type,
		Note:     request.Note,
		ActorID:  currentUser(c).ID,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentProduct(product))
}

// handleListProductInventory devuelve el historial de movimientos de un producto.
func (a *API) handleListProductInventory(c *fiber.Ctx) error {
	movements, err := a.inventory.ListMovements(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentInventory(movements))
}
