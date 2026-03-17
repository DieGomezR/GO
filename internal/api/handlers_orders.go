// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"net/http"

	"tienda-go/internal/service"
)

// createOrderItemRequest representa un item enviado por el cliente.
type createOrderItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int64  `json:"quantity"`
}

// createOrderRequest representa el body para registrar una venta.
type createOrderRequest struct {
	CustomerName string                   `json:"customer_name"`
	Items        []createOrderItemRequest `json:"items"`
}

// handleCreateOrder transforma el request HTTP en un caso de uso de venta.
func (a *API) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var request createOrderRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, err)
		return
	}

	items := make([]service.CreateOrderItemInput, 0, len(request.Items))
	for _, item := range request.Items {
		items = append(items, service.CreateOrderItemInput{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
	}

	order, err := a.orders.CreateOrder(service.CreateOrderInput{
		CustomerName: request.CustomerName,
		CashierID:    currentUser(r).ID,
		Items:        items,
	})
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusCreated, presentOrder(order))
}

// handleListOrders devuelve el historial de ventas.
func (a *API) handleListOrders(w http.ResponseWriter, _ *http.Request) {
	orders, err := a.orders.ListOrders()
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentOrders(orders))
}

// handleGetOrder devuelve una orden por ID.
func (a *API) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	order, err := a.orders.GetOrder(r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}

	writeData(w, http.StatusOK, presentOrder(order))
}
