// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"net/http"

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
func (a *API) handleCreateProduct(w http.ResponseWriter, r *http.Request) {
	var request createProductRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	product, err := a.products.CreateProduct(service.CreateProductInput{
		SKU:          request.SKU,
		Name:         request.Name,
		Description:  request.Description,
		PriceCents:   request.PriceCents,
		InitialStock: request.InitialStock,
		ActorID:      currentUser(r).ID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusCreated, presentProduct(product))
}

// handleUpdateProduct edita atributos del producto sin tocar el historial.
func (a *API) handleUpdateProduct(w http.ResponseWriter, r *http.Request) {
	var request updateProductRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	product, err := a.products.UpdateProduct(r.PathValue("id"), service.UpdateProductInput{
		SKU:         request.SKU,
		Name:        request.Name,
		Description: request.Description,
		PriceCents:  request.PriceCents,
		Active:      request.Active,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentProduct(product))
}

// handleListProducts devuelve el catalogo completo.
func (a *API) handleListProducts(w http.ResponseWriter, _ *http.Request) {
	products, err := a.products.ListProducts()
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentProducts(products))
}

// handleGetProduct devuelve un producto puntual.
func (a *API) handleGetProduct(w http.ResponseWriter, r *http.Request) {
	product, err := a.products.GetProduct(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentProduct(product))
}

// handleAdjustInventory registra entradas o correcciones de stock.
func (a *API) handleAdjustInventory(w http.ResponseWriter, r *http.Request) {
	var request adjustInventoryRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	product, err := a.inventory.AdjustStock(r.PathValue("id"), service.AdjustStockInput{
		Quantity: request.Quantity,
		Type:     request.Type,
		Note:     request.Note,
		ActorID:  currentUser(r).ID,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentProduct(product))
}

// handleListProductInventory devuelve el historial de movimientos de un producto.
func (a *API) handleListProductInventory(w http.ResponseWriter, r *http.Request) {
	movements, err := a.inventory.ListMovements(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentInventory(movements))
}
