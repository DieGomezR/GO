// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/config"
	"tienda-go/internal/domain"
	"tienda-go/internal/service"
)

// API reune dependencias compartidas por handlers y middleware.
type API struct {
	cfg       config.Config
	logger    *slog.Logger
	auth      *service.AuthService
	users     *service.UserService
	products  *service.ProductService
	inventory *service.InventoryService
	orders    *service.OrderService
	dashboard *service.DashboardService
	partner   func(router fiber.Router)
}

// New construye la instancia principal de la capa HTTP.
func New(
	cfg config.Config,
	logger *slog.Logger,
	auth *service.AuthService,
	users *service.UserService,
	products *service.ProductService,
	inventory *service.InventoryService,
	orders *service.OrderService,
	dashboard *service.DashboardService,
) *API {
	return &API{
		cfg:       cfg,
		logger:    logger,
		auth:      auth,
		users:     users,
		products:  products,
		inventory: inventory,
		orders:    orders,
		dashboard: dashboard,
	}
}

// RegisterRoutes registra las rutas del modulo principal y las opcionales.
func (a *API) RegisterRoutes(app *fiber.App) {
	// Rutas publicas.
	app.Get("/healthz", a.handleHealth)
	app.Post("/v1/auth/login", a.handleLogin)

	// Rutas autenticadas comunes.
	app.Post("/v1/auth/logout", a.authenticated(), a.handleLogout)
	app.Get("/v1/me", a.authenticated(), a.handleMe)

	// Gestion de usuarios y resumen administrativo.
	app.Get("/v1/users", a.requireRoles(domain.RoleAdmin, domain.RoleManager), a.handleListUsers)
	app.Post("/v1/users", a.requireRoles(domain.RoleAdmin), a.handleCreateUser)
	app.Get("/v1/dashboard/summary", a.requireRoles(domain.RoleAdmin, domain.RoleManager), a.handleDashboardSummary)

	// Catalogo e inventario.
	app.Get("/v1/products", a.authenticated(), a.handleListProducts)
	app.Get("/v1/products/:id", a.authenticated(), a.handleGetProduct)
	app.Post("/v1/products", a.requireRoles(domain.RoleAdmin, domain.RoleManager), a.handleCreateProduct)
	app.Put("/v1/products/:id", a.requireRoles(domain.RoleAdmin, domain.RoleManager), a.handleUpdateProduct)
	app.Get("/v1/products/:id/inventory", a.authenticated(), a.handleListProductInventory)
	app.Post("/v1/products/:id/inventory", a.requireRoles(domain.RoleAdmin, domain.RoleManager), a.handleAdjustInventory)

	// Ventas.
	app.Get("/v1/orders", a.authenticated(), a.handleListOrders)
	app.Get("/v1/orders/:id", a.authenticated(), a.handleGetOrder)
	app.Post("/v1/orders", a.requireRoles(domain.RoleAdmin, domain.RoleManager, domain.RoleCashier), a.handleCreateOrder)

	// Port del controller PHP legado, ahora montado tambien sobre Fiber.
	if a.partner != nil {
		a.partner(app.Group("/v1/partner-users", a.authenticated()))
	}
}

// App construye la instancia Fiber completa con middlewares globales.
func (a *API) App() *fiber.App {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ReadTimeout:           a.cfg.ReadTimeout,
		WriteTimeout:          a.cfg.WriteTimeout,
	})

	app.Use(a.logging())
	app.Use(a.recoverer())
	a.RegisterRoutes(app)
	app.Use(func(c *fiber.Ctx) error {
		return writeJSON(c, fiber.StatusNotFound, map[string]any{
			"error": "not found",
		})
	})

	return app
}

// SetPartnerRoutes monta un subrouter adicional para el port de partner users.
func (a *API) SetPartnerRoutes(register func(router fiber.Router)) {
	a.partner = register
}
