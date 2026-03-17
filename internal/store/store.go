// Package store contiene una implementacion en memoria para persistir datos
// durante la vida del proceso. Es util para aprender y para challenges.
package store

import (
	"cmp"
	"slices"
	"strings"
	"sync"

	"tienda-go/internal/domain"
)

// MemoryStore agrupa todos los recursos persistidos en memoria.
// Un solo mutex protege la estructura para mantener el ejemplo sencillo.
type MemoryStore struct {
	mu            sync.RWMutex
	opMu          sync.Mutex
	users         map[string]domain.User
	usersByEmail  map[string]string
	products      map[string]domain.Product
	productsBySKU map[string]string
	orders        map[string]domain.Order
	movements     map[string][]domain.InventoryMovement
	sessions      map[string]domain.Session
}

// NewMemoryStore inicializa todos los mapas internos del almacenamiento.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:         make(map[string]domain.User),
		usersByEmail:  make(map[string]string),
		products:      make(map[string]domain.Product),
		productsBySKU: make(map[string]string),
		orders:        make(map[string]domain.Order),
		movements:     make(map[string][]domain.InventoryMovement),
		sessions:      make(map[string]domain.Session),
	}
}

// WithLock serializa operaciones de varios pasos dentro del mismo proceso.
func (s *MemoryStore) WithLock(fn func() error) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	return fn()
}

// CreateUser inserta un usuario y garantiza email unico.
func (s *MemoryStore) CreateUser(user domain.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	email := strings.ToLower(user.Email)
	if _, exists := s.usersByEmail[email]; exists {
		return domain.ErrConflict
	}

	s.users[user.ID] = user
	s.usersByEmail[email] = user.ID

	return nil
}

// GetUserByEmail resuelve un usuario a partir de su email normalizado.
func (s *MemoryStore) GetUserByEmail(email string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	id, ok := s.usersByEmail[strings.ToLower(email)]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}

	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}

	return user, nil
}

// GetUserByID devuelve un usuario por su ID.
func (s *MemoryStore) GetUserByID(id string) (domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}

	return user, nil
}

// ListUsers devuelve usuarios ordenados por fecha de creacion.
func (s *MemoryStore) ListUsers() ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]domain.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	slices.SortFunc(users, func(a, b domain.User) int {
		return cmp.Compare(a.CreatedAt.UnixNano(), b.CreatedAt.UnixNano())
	})

	return users, nil
}

// CreateSession guarda un token de autenticacion.
func (s *MemoryStore) CreateSession(session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions[session.Token] = session
	return nil
}

// GetSession busca una sesion existente.
func (s *MemoryStore) GetSession(token string) (domain.Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.sessions[token]
	if !ok {
		return domain.Session{}, domain.ErrUnauthorized
	}

	return session, nil
}

// DeleteSession elimina una sesion; si no existe, la operacion sigue siendo idempotente.
func (s *MemoryStore) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
	return nil
}

// CreateProduct inserta un producto garantizando SKU unico.
func (s *MemoryStore) CreateProduct(product domain.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sku := strings.ToLower(product.SKU)
	if _, exists := s.productsBySKU[sku]; exists {
		return domain.ErrConflict
	}

	s.products[product.ID] = product
	s.productsBySKU[sku] = product.ID
	return nil
}

// UpdateProduct reemplaza el producto completo y mantiene consistente el indice por SKU.
func (s *MemoryStore) UpdateProduct(product domain.Product) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.products[product.ID]
	if !ok {
		return domain.ErrNotFound
	}

	previousSKU := strings.ToLower(existing.SKU)
	currentSKU := strings.ToLower(product.SKU)
	if previousSKU != currentSKU {
		if id, exists := s.productsBySKU[currentSKU]; exists && id != product.ID {
			return domain.ErrConflict
		}

		delete(s.productsBySKU, previousSKU)
		s.productsBySKU[currentSKU] = product.ID
	}

	s.products[product.ID] = product
	return nil
}

// GetProduct devuelve un producto por ID.
func (s *MemoryStore) GetProduct(id string) (domain.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	product, ok := s.products[id]
	if !ok {
		return domain.Product{}, domain.ErrNotFound
	}

	return product, nil
}

// ListProducts devuelve el catalogo ordenado alfabeticamente por nombre.
func (s *MemoryStore) ListProducts() ([]domain.Product, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	products := make([]domain.Product, 0, len(s.products))
	for _, product := range s.products {
		products = append(products, product)
	}

	slices.SortFunc(products, func(a, b domain.Product) int {
		return cmp.Compare(a.Name, b.Name)
	})

	return products, nil
}

// AddMovement agrega un movimiento al historial del producto.
func (s *MemoryStore) AddMovement(movement domain.InventoryMovement) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.movements[movement.ProductID] = append(s.movements[movement.ProductID], movement)
	return nil
}

// ListProductMovements devuelve los movimientos del mas reciente al mas antiguo.
// Como siempre se insertan cronologicamente, recorrer al reves evita un sort extra.
func (s *MemoryStore) ListProductMovements(productID string) ([]domain.InventoryMovement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	source := s.movements[productID]
	movements := make([]domain.InventoryMovement, 0, len(source))
	for index := len(source) - 1; index >= 0; index-- {
		movements = append(movements, source[index])
	}

	return movements, nil
}

// CreateOrder persiste una orden copiando el slice de items para no compartir memoria.
func (s *MemoryStore) CreateOrder(order domain.Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.orders[order.ID] = cloneOrder(order)
	return nil
}

// GetOrder devuelve una copia segura de la orden.
func (s *MemoryStore) GetOrder(id string) (domain.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	order, ok := s.orders[id]
	if !ok {
		return domain.Order{}, domain.ErrNotFound
	}

	return cloneOrder(order), nil
}

// ListOrders lista las ordenes de la mas reciente a la mas antigua.
func (s *MemoryStore) ListOrders() ([]domain.Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	orders := make([]domain.Order, 0, len(s.orders))
	for _, order := range s.orders {
		orders = append(orders, cloneOrder(order))
	}

	slices.SortFunc(orders, func(a, b domain.Order) int {
		return cmp.Compare(b.CreatedAt.UnixNano(), a.CreatedAt.UnixNano())
	})

	return orders, nil
}

// cloneOrder clona el slice de items para evitar mutaciones accidentales desde fuera.
func cloneOrder(order domain.Order) domain.Order {
	order.Items = append([]domain.OrderItem(nil), order.Items...)
	return order
}
