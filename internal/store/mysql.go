package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"tienda-go/internal/domain"
)

// MySQLStore persiste la informacion de la tienda en una base de datos MySQL.
type MySQLStore struct {
	db   *sql.DB
	opMu sync.Mutex
}

// NewMySQLStore abre la conexion y, si se solicita, crea el esquema basico.
func NewMySQLStore(dsn string, autoMigrate bool) (*MySQLStore, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, fmt.Errorf("mysql dsn is required")
	}

	normalized, err := normalizeMySQLDSN(dsn)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("mysql", normalized)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql connection: %w", err)
	}

	if autoMigrate {
		if err := migrateMySQLSchema(db); err != nil {
			_ = db.Close()
			return nil, err
		}
	}

	return &MySQLStore{db: db}, nil
}

// Close libera la conexion abierta con MySQL.
func (s *MySQLStore) Close() error {
	return s.db.Close()
}

// WithLock serializa operaciones de varios pasos dentro del mismo proceso.
func (s *MySQLStore) WithLock(fn func() error) error {
	s.opMu.Lock()
	defer s.opMu.Unlock()

	return fn()
}

// CreateUser inserta un usuario asegurando email unico.
func (s *MySQLStore) CreateUser(user domain.User) error {
	const query = `
		INSERT INTO users
			(id, name, email, role, password_salt, password_hash, active, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		user.ID,
		user.Name,
		strings.ToLower(user.Email),
		string(user.Role),
		user.PasswordSalt,
		user.PasswordHash,
		user.Active,
		user.CreatedAt.UTC(),
	)
	return mapMySQLError(err)
}

// GetUserByEmail busca un usuario a partir de su email normalizado.
func (s *MySQLStore) GetUserByEmail(email string) (domain.User, error) {
	const query = `
		SELECT id, name, email, role, password_salt, password_hash, active, created_at
		FROM users
		WHERE email = ?
		LIMIT 1`

	row := s.db.QueryRow(query, strings.ToLower(email))
	return scanUser(row)
}

// GetUserByID devuelve un usuario por su identificador.
func (s *MySQLStore) GetUserByID(id string) (domain.User, error) {
	const query = `
		SELECT id, name, email, role, password_salt, password_hash, active, created_at
		FROM users
		WHERE id = ?
		LIMIT 1`

	row := s.db.QueryRow(query, id)
	return scanUser(row)
}

// ListUsers lista usuarios ordenados por fecha de creacion.
func (s *MySQLStore) ListUsers() ([]domain.User, error) {
	const query = `
		SELECT id, name, email, role, password_salt, password_hash, active, created_at
		FROM users
		ORDER BY created_at ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]domain.User, 0)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, rows.Err()
}

// CreateSession guarda una sesion autenticada.
func (s *MySQLStore) CreateSession(session domain.Session) error {
	const query = `
		INSERT INTO sessions (token, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		session.Token,
		session.UserID,
		session.ExpiresAt.UTC(),
		session.CreatedAt.UTC(),
	)
	return mapMySQLError(err)
}

// GetSession resuelve un token a su sesion persistida.
func (s *MySQLStore) GetSession(token string) (domain.Session, error) {
	const query = `
		SELECT token, user_id, expires_at, created_at
		FROM sessions
		WHERE token = ?
		LIMIT 1`

	row := s.db.QueryRow(query, token)
	var session domain.Session
	if err := row.Scan(&session.Token, &session.UserID, &session.ExpiresAt, &session.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Session{}, domain.ErrUnauthorized
		}
		return domain.Session{}, err
	}
	return session, nil
}

// DeleteSession elimina una sesion si existe.
func (s *MySQLStore) DeleteSession(token string) error {
	const query = `DELETE FROM sessions WHERE token = ?`
	_, err := s.db.Exec(query, token)
	return err
}

// CreateProduct inserta un producto garantizando SKU unico.
func (s *MySQLStore) CreateProduct(product domain.Product) error {
	const query = `
		INSERT INTO products
			(id, sku, name, description, price_cents, stock, active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		product.ID,
		product.SKU,
		product.Name,
		product.Description,
		product.PriceCents,
		product.Stock,
		product.Active,
		product.CreatedAt.UTC(),
		product.UpdatedAt.UTC(),
	)
	return mapMySQLError(err)
}

// UpdateProduct reemplaza el estado actual de un producto.
func (s *MySQLStore) UpdateProduct(product domain.Product) error {
	const query = `
		UPDATE products
		SET sku = ?, name = ?, description = ?, price_cents = ?, stock = ?, active = ?, updated_at = ?
		WHERE id = ?`

	result, err := s.db.Exec(
		query,
		product.SKU,
		product.Name,
		product.Description,
		product.PriceCents,
		product.Stock,
		product.Active,
		product.UpdatedAt.UTC(),
		product.ID,
	)
	if err != nil {
		return mapMySQLError(err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetProduct devuelve un producto por ID.
func (s *MySQLStore) GetProduct(id string) (domain.Product, error) {
	const query = `
		SELECT id, sku, name, description, price_cents, stock, active, created_at, updated_at
		FROM products
		WHERE id = ?
		LIMIT 1`

	row := s.db.QueryRow(query, id)
	return scanProduct(row)
}

// ListProducts devuelve el catalogo ordenado alfabeticamente.
func (s *MySQLStore) ListProducts() ([]domain.Product, error) {
	const query = `
		SELECT id, sku, name, description, price_cents, stock, active, created_at, updated_at
		FROM products
		ORDER BY name ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := make([]domain.Product, 0)
	for rows.Next() {
		product, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, product)
	}

	return products, rows.Err()
}

// AddMovement agrega un movimiento de inventario.
func (s *MySQLStore) AddMovement(movement domain.InventoryMovement) error {
	const query = `
		INSERT INTO inventory_movements
			(id, product_id, type, quantity, note, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(
		query,
		movement.ID,
		movement.ProductID,
		string(movement.Type),
		movement.Quantity,
		movement.Note,
		movement.CreatedBy,
		movement.CreatedAt.UTC(),
	)
	return mapMySQLError(err)
}

// ListProductMovements devuelve el historial del producto del mas reciente al mas antiguo.
func (s *MySQLStore) ListProductMovements(productID string) ([]domain.InventoryMovement, error) {
	const query = `
		SELECT id, product_id, type, quantity, note, created_by, created_at
		FROM inventory_movements
		WHERE product_id = ?
		ORDER BY created_at DESC, id DESC`

	rows, err := s.db.Query(query, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	movements := make([]domain.InventoryMovement, 0)
	for rows.Next() {
		var movement domain.InventoryMovement
		var movementType string
		if err := rows.Scan(
			&movement.ID,
			&movement.ProductID,
			&movementType,
			&movement.Quantity,
			&movement.Note,
			&movement.CreatedBy,
			&movement.CreatedAt,
		); err != nil {
			return nil, err
		}
		movement.Type = domain.MovementType(movementType)
		movements = append(movements, movement)
	}

	return movements, rows.Err()
}

// CreateOrder persiste la cabecera y los items de una venta.
func (s *MySQLStore) CreateOrder(order domain.Order) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	const insertOrder = `
		INSERT INTO orders
			(id, number, customer_name, cashier_id, status, total_cents, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	if _, err = tx.Exec(
		insertOrder,
		order.ID,
		order.Number,
		order.CustomerName,
		order.CashierID,
		string(order.Status),
		order.TotalCents,
		order.CreatedAt.UTC(),
	); err != nil {
		return mapMySQLError(err)
	}

	const insertItem = `
		INSERT INTO order_items
			(order_id, product_id, product_name, quantity, unit_price_cents, subtotal_cents)
		VALUES (?, ?, ?, ?, ?, ?)`

	for _, item := range order.Items {
		if _, err = tx.Exec(
			insertItem,
			order.ID,
			item.ProductID,
			item.ProductName,
			item.Quantity,
			item.UnitPriceCents,
			item.SubtotalCents,
		); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// GetOrder devuelve una orden por ID junto con sus items.
func (s *MySQLStore) GetOrder(id string) (domain.Order, error) {
	const query = `
		SELECT id, number, customer_name, cashier_id, status, total_cents, created_at
		FROM orders
		WHERE id = ?
		LIMIT 1`

	row := s.db.QueryRow(query, id)
	order, err := scanOrder(row)
	if err != nil {
		return domain.Order{}, err
	}

	items, err := s.listOrderItems(order.ID)
	if err != nil {
		return domain.Order{}, err
	}
	order.Items = items

	return cloneOrder(order), nil
}

// ListOrders lista las ventas de la mas reciente a la mas antigua.
func (s *MySQLStore) ListOrders() ([]domain.Order, error) {
	const query = `
		SELECT id, number, customer_name, cashier_id, status, total_cents, created_at
		FROM orders
		ORDER BY created_at DESC, id DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}

		items, err := s.listOrderItems(order.ID)
		if err != nil {
			return nil, err
		}
		order.Items = items
		orders = append(orders, cloneOrder(order))
	}

	return orders, rows.Err()
}

func (s *MySQLStore) listOrderItems(orderID string) ([]domain.OrderItem, error) {
	const query = `
		SELECT product_id, product_name, quantity, unit_price_cents, subtotal_cents
		FROM order_items
		WHERE order_id = ?
		ORDER BY id ASC`

	rows, err := s.db.Query(query, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.OrderItem, 0)
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(
			&item.ProductID,
			&item.ProductName,
			&item.Quantity,
			&item.UnitPriceCents,
			&item.SubtotalCents,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

func scanUser(scanner interface{ Scan(dest ...any) error }) (domain.User, error) {
	var user domain.User
	var role string
	if err := scanner.Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&role,
		&user.PasswordSalt,
		&user.PasswordHash,
		&user.Active,
		&user.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}

	user.Role = domain.Role(role)
	return user, nil
}

func scanProduct(scanner interface{ Scan(dest ...any) error }) (domain.Product, error) {
	var product domain.Product
	if err := scanner.Scan(
		&product.ID,
		&product.SKU,
		&product.Name,
		&product.Description,
		&product.PriceCents,
		&product.Stock,
		&product.Active,
		&product.CreatedAt,
		&product.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Product{}, domain.ErrNotFound
		}
		return domain.Product{}, err
	}

	return product, nil
}

func scanOrder(scanner interface{ Scan(dest ...any) error }) (domain.Order, error) {
	var order domain.Order
	var status string
	if err := scanner.Scan(
		&order.ID,
		&order.Number,
		&order.CustomerName,
		&order.CashierID,
		&status,
		&order.TotalCents,
		&order.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Order{}, domain.ErrNotFound
		}
		return domain.Order{}, err
	}

	order.Status = domain.OrderStatus(status)
	return order, nil
}

func mapMySQLError(err error) error {
	if err == nil {
		return nil
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		return domain.ErrConflict
	}

	return err
}

func normalizeMySQLDSN(raw string) (string, error) {
	cfg, err := mysql.ParseDSN(raw)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}

	cfg.ParseTime = true
	cfg.Loc = time.UTC
	if cfg.Collation == "" {
		cfg.Collation = "utf8mb4_unicode_ci"
	}

	return cfg.FormatDSN(), nil
}
