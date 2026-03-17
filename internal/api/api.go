// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"log/slog"
	"net/http"

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
	partner   http.Handler
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
func (a *API) RegisterRoutes(mux *http.ServeMux) {
	// Rutas publicas.
	mux.HandleFunc("GET /healthz", a.handleHealth)
	mux.HandleFunc("POST /v1/auth/login", a.handleLogin)

	// Rutas autenticadas comunes.
	mux.Handle("POST /v1/auth/logout", a.authenticated(http.HandlerFunc(a.handleLogout)))
	mux.Handle("GET /v1/me", a.authenticated(http.HandlerFunc(a.handleMe)))

	// Gestion de usuarios y resumen administrativo.
	mux.Handle("GET /v1/users", a.requireRoles(domain.RoleAdmin, domain.RoleManager)(http.HandlerFunc(a.handleListUsers)))
	mux.Handle("POST /v1/users", a.requireRoles(domain.RoleAdmin)(http.HandlerFunc(a.handleCreateUser)))

	mux.Handle("GET /v1/dashboard/summary", a.requireRoles(domain.RoleAdmin, domain.RoleManager)(http.HandlerFunc(a.handleDashboardSummary)))

	// Catalogo e inventario.
	mux.Handle("GET /v1/products", a.authenticated(http.HandlerFunc(a.handleListProducts)))
	mux.Handle("GET /v1/products/{id}", a.authenticated(http.HandlerFunc(a.handleGetProduct)))
	mux.Handle("POST /v1/products", a.requireRoles(domain.RoleAdmin, domain.RoleManager)(http.HandlerFunc(a.handleCreateProduct)))
	mux.Handle("PUT /v1/products/{id}", a.requireRoles(domain.RoleAdmin, domain.RoleManager)(http.HandlerFunc(a.handleUpdateProduct)))
	mux.Handle("GET /v1/products/{id}/inventory", a.authenticated(http.HandlerFunc(a.handleListProductInventory)))
	mux.Handle("POST /v1/products/{id}/inventory", a.requireRoles(domain.RoleAdmin, domain.RoleManager)(http.HandlerFunc(a.handleAdjustInventory)))

	// Ventas.
	mux.Handle("GET /v1/orders", a.authenticated(http.HandlerFunc(a.handleListOrders)))
	mux.Handle("GET /v1/orders/{id}", a.authenticated(http.HandlerFunc(a.handleGetOrder)))
	mux.Handle("POST /v1/orders", a.requireRoles(domain.RoleAdmin, domain.RoleManager, domain.RoleCashier)(http.HandlerFunc(a.handleCreateOrder)))

	// Port del controller PHP legado.
	if a.partner != nil {
		mux.Handle("/v1/partner-users/", a.authenticated(a.partner))
	}
}

// Router registra todas las rutas y compone los middlewares globales.
func (a *API) Router() http.Handler {
	mux := http.NewServeMux()
	a.RegisterRoutes(mux)
	return a.Wrap(mux)
}

// Wrap aplica los middlewares globales de la API sobre cualquier handler.
func (a *API) Wrap(next http.Handler) http.Handler {
	return a.logging(a.recoverer(next))
}

// SetPartnerRoutes monta un subrouter adicional para el port de partner users.
func (a *API) SetPartnerRoutes(handler http.Handler) {
	a.partner = handler
}
