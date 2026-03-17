// Package bootstrap contiene tareas de arranque como la carga de datos iniciales.
package bootstrap

import (
	"tienda-go/internal/domain"
	"tienda-go/internal/service"
)

// SeedCredentials conserva las credenciales demo para mostrarlas en logs al arrancar.
type SeedCredentials struct {
	Name     string
	Email    string
	Password string
	Role     domain.Role
}

// SeedData agrupa la informacion generada durante el bootstrap.
type SeedData struct {
	Users []SeedCredentials
}

// Seed crea usuarios y productos base para que la API sea util desde el primer arranque.
func Seed(users *service.UserService, products *service.ProductService) (SeedData, error) {
	existingUsers, err := users.ListUsers()
	if err != nil {
		return SeedData{}, err
	}
	if len(existingUsers) > 0 {
		return SeedData{}, nil
	}

	existingProducts, err := products.ListProducts()
	if err != nil {
		return SeedData{}, err
	}
	if len(existingProducts) > 0 {
		return SeedData{}, nil
	}

	seedUsers := []SeedCredentials{
		{Name: "Admin Principal", Email: "admin@store.local", Password: "Admin1234!", Role: domain.RoleAdmin},
		{Name: "Gerente Tienda", Email: "manager@store.local", Password: "Manager1234!", Role: domain.RoleManager},
		{Name: "Caja Demo", Email: "cashier@store.local", Password: "Cashier1234!", Role: domain.RoleCashier},
	}

	createdUsers := make([]domain.User, 0, len(seedUsers))
	for _, user := range seedUsers {
		// Se reutiliza el servicio real para sembrar datos siguiendo las mismas reglas de negocio.
		created, err := users.CreateUser(service.CreateUserInput{
			Name:     user.Name,
			Email:    user.Email,
			Password: user.Password,
			Role:     user.Role,
		})
		if err != nil {
			return SeedData{}, err
		}

		createdUsers = append(createdUsers, created)
	}

	// El admin sembrado actua como responsable de la carga inicial del catalogo.
	adminID := createdUsers[0].ID
	productsToCreate := []service.CreateProductInput{
		{
			SKU:          "KB-001",
			Name:         "Teclado Mecanico",
			Description:  "Teclado para escritorio con switches azules.",
			PriceCents:   2599000,
			InitialStock: 15,
			ActorID:      adminID,
		},
		{
			SKU:          "MS-002",
			Name:         "Mouse Inalambrico",
			Description:  "Mouse ergonomico recargable para oficina.",
			PriceCents:   1199000,
			InitialStock: 25,
			ActorID:      adminID,
		},
		{
			SKU:          "MN-003",
			Name:         "Monitor 27 pulgadas",
			Description:  "Monitor IPS 2K para trabajo y gaming casual.",
			PriceCents:   8999000,
			InitialStock: 8,
			ActorID:      adminID,
		},
		{
			SKU:          "HB-004",
			Name:         "Hub USB-C",
			Description:  "Hub multipuerto con HDMI, USB-A y lector SD.",
			PriceCents:   3499000,
			InitialStock: 12,
			ActorID:      adminID,
		},
	}

	for _, product := range productsToCreate {
		if _, err := products.CreateProduct(product); err != nil {
			return SeedData{}, err
		}
	}

	return SeedData{Users: seedUsers}, nil
}
