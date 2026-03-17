package mysqlrepo

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"tienda-go/internal/partnerusers"
)

// Repositories agrupa implementaciones concretas de los puertos contra MySQL.
type Repositories struct {
	AppDB     *sql.DB
	PartnerDB *sql.DB

	Users       partnerusers.UserRepository
	ISPs        partnerusers.ISPRepository
	Subscribers partnerusers.SubscriberRepository
	Audits      partnerusers.EarlyDeactivationAuditRepository
}

// Open inicializa ambas conexiones MySQL y construye los repositorios listos para inyectar.
func Open(cfg Config) (*Repositories, error) {
	cfg = cfg.WithDefaults()
	if cfg.AppDSN == "" {
		return nil, fmt.Errorf("mysqlrepo: AppDSN is required")
	}
	if cfg.PartnerDSN == "" {
		return nil, fmt.Errorf("mysqlrepo: PartnerDSN is required")
	}

	appDB, err := sql.Open("mysql", cfg.AppDSN)
	if err != nil {
		return nil, fmt.Errorf("mysqlrepo: open app db: %w", err)
	}

	partnerDB, err := sql.Open("mysql", cfg.PartnerDSN)
	if err != nil {
		_ = appDB.Close()
		return nil, fmt.Errorf("mysqlrepo: open partner db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := appDB.PingContext(ctx); err != nil {
		_ = appDB.Close()
		_ = partnerDB.Close()
		return nil, fmt.Errorf("mysqlrepo: ping app db: %w", err)
	}

	if err := partnerDB.PingContext(ctx); err != nil {
		_ = appDB.Close()
		_ = partnerDB.Close()
		return nil, fmt.Errorf("mysqlrepo: ping partner db: %w", err)
	}

	return &Repositories{
		AppDB:       appDB,
		PartnerDB:   partnerDB,
		Users:       &UserRepository{db: appDB, table: cfg.UsersTable},
		ISPs:        &ISPRepository{db: partnerDB, table: cfg.ISPTable},
		Subscribers: &SubscriberRepository{db: partnerDB},
		Audits:      &AuditRepository{db: partnerDB, table: cfg.EarlyAuditTable},
	}, nil
}

// OpenPartnerOnly inicializa solo la conexion a db_prod para endpoints de
// consulta que no dependen de la base app/mysql.
func OpenPartnerOnly(cfg Config) (*Repositories, error) {
	cfg = cfg.WithDefaults()
	if cfg.PartnerDSN == "" {
		return nil, fmt.Errorf("mysqlrepo: PartnerDSN is required")
	}

	partnerDB, err := sql.Open("mysql", cfg.PartnerDSN)
	if err != nil {
		return nil, fmt.Errorf("mysqlrepo: open partner db: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	if err := partnerDB.PingContext(ctx); err != nil {
		_ = partnerDB.Close()
		return nil, fmt.Errorf("mysqlrepo: ping partner db: %w", err)
	}

	return &Repositories{
		PartnerDB:   partnerDB,
		ISPs:        &ISPRepository{db: partnerDB, table: cfg.ISPTable},
		Subscribers: &SubscriberRepository{db: partnerDB},
		Audits:      &AuditRepository{db: partnerDB, table: cfg.EarlyAuditTable},
	}, nil
}

// Close libera ambas conexiones.
func (r *Repositories) Close() error {
	var firstErr error
	if r.AppDB != nil {
		if err := r.AppDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if r.PartnerDB != nil {
		if err := r.PartnerDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
