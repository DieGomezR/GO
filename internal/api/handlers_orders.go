// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"github.com/gofiber/fiber/v2"
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
func (a *API) handleCreateOrder(c *fiber.Ctx) error {
	var request createOrderRequest
	if err := decodeJSON(c, &request); err != nil {
		return writeError(c, err)
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
		CashierID:    currentUser(c).ID,
		Items:        items,
	})
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusCreated, presentOrder(order))
}

// handleListOrders devuelve el historial de ventas.
func (a *API) handleListOrders(c *fiber.Ctx) error {
	orders, err := a.orders.ListOrders()
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentOrders(orders))
}

// handleGetOrder devuelve una orden por ID.
func (a *API) handleGetOrder(c *fiber.Ctx) error {
	order, err := a.orders.GetOrder(c.Params("id"))
	if err != nil {
		return writeError(c, err)
	}

	return writeData(c, fiber.StatusOK, presentOrder(order))
}
