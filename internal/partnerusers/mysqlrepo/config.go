// Package mysqlrepo implementa los puertos de partnerusers usando MySQL.
package mysqlrepo

import "time"

// Config define las dos conexiones que usaba el controller original:
// una para la base "app/mysql" y otra para "db_prod".
type Config struct {
	AppDSN     string
	PartnerDSN string

	UsersTable      string
	ISPTable        string
	EarlyAuditTable string

	ConnectTimeout time.Duration
}

// WithDefaults aplica nombres de tabla razonables para el port actual.
func (c Config) WithDefaults() Config {
	if c.UsersTable == "" {
		c.UsersTable = "Users"
	}
	if c.ISPTable == "" {
		c.ISPTable = "isps"
	}
	if c.EarlyAuditTable == "" {
		c.EarlyAuditTable = "early_deactivation_audits"
	}
	if c.ConnectTimeout <= 0 {
		c.ConnectTimeout = 5 * time.Second
	}
	return c
}
