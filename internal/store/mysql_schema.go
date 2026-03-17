package store

import (
	"database/sql"
	"fmt"
)

// mysqlSchemaStatements mantiene el esquema minimo para la tienda.
var mysqlSchemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		email VARCHAR(255) NOT NULL,
		role VARCHAR(32) NOT NULL,
		password_salt VARCHAR(255) NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at DATETIME(6) NOT NULL,
		UNIQUE KEY uq_users_email (email)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS sessions (
		token VARCHAR(64) NOT NULL PRIMARY KEY,
		user_id VARCHAR(64) NOT NULL,
		expires_at DATETIME(6) NOT NULL,
		created_at DATETIME(6) NOT NULL,
		KEY idx_sessions_user_id (user_id),
		KEY idx_sessions_expires_at (expires_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS products (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		sku VARCHAR(64) NOT NULL,
		name VARCHAR(255) NOT NULL,
		description TEXT NOT NULL,
		price_cents BIGINT NOT NULL,
		stock BIGINT NOT NULL,
		active BOOLEAN NOT NULL DEFAULT TRUE,
		created_at DATETIME(6) NOT NULL,
		updated_at DATETIME(6) NOT NULL,
		UNIQUE KEY uq_products_sku (sku),
		KEY idx_products_name (name)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS inventory_movements (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		product_id VARCHAR(64) NOT NULL,
		type VARCHAR(32) NOT NULL,
		quantity BIGINT NOT NULL,
		note TEXT NOT NULL,
		created_by VARCHAR(64) NOT NULL,
		created_at DATETIME(6) NOT NULL,
		KEY idx_inventory_movements_product_created (product_id, created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS orders (
		id VARCHAR(64) NOT NULL PRIMARY KEY,
		number VARCHAR(64) NOT NULL,
		customer_name VARCHAR(255) NOT NULL,
		cashier_id VARCHAR(64) NOT NULL,
		status VARCHAR(32) NOT NULL,
		total_cents BIGINT NOT NULL,
		created_at DATETIME(6) NOT NULL,
		UNIQUE KEY uq_orders_number (number),
		KEY idx_orders_created_at (created_at)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
	`CREATE TABLE IF NOT EXISTS order_items (
		id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		order_id VARCHAR(64) NOT NULL,
		product_id VARCHAR(64) NOT NULL,
		product_name VARCHAR(255) NOT NULL,
		quantity BIGINT NOT NULL,
		unit_price_cents BIGINT NOT NULL,
		subtotal_cents BIGINT NOT NULL,
		KEY idx_order_items_order_id (order_id),
		CONSTRAINT fk_order_items_order_id FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`,
}

func migrateMySQLSchema(db *sql.DB) error {
	for _, statement := range mysqlSchemaStatements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("migrate mysql schema: %w", err)
		}
	}

	return nil
}
