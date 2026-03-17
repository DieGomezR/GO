// Package config centraliza la carga de variables de entorno para que el resto
// del proyecto no tenga que leer os.Getenv directamente.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config concentra toda la configuracion operativa de la aplicacion.
type Config struct {
	// AppName se usa en logs y respuestas simples como /healthz.
	AppName string
	// Env ayuda a distinguir el ambiente de ejecucion.
	Env string
	// Address es la direccion donde escucha el servidor HTTP.
	Address string
	// ReadTimeout y WriteTimeout protegen al servidor de clientes lentos.
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	// ShutdownWait da tiempo para cerrar conexiones abiertas.
	ShutdownWait time.Duration
	// TokenTTL define cuanto vive una sesion autenticada.
	TokenTTL time.Duration
	// LowStockLimit alimenta el dashboard para alertar productos criticos.
	LowStockLimit int64
	// DefaultCurrency queda lista para futuras extensiones del proyecto.
	DefaultCurrency string
	// StoreDriver define si la app usa memoria o MySQL para persistir.
	StoreDriver string
	// MySQLDSN contiene la cadena de conexion hacia la base de datos.
	MySQLDSN string
	// MySQLAutoMigrate crea el esquema minimo al arrancar si no existe.
	MySQLAutoMigrate bool
	// SeedOnStart permite sembrar datos demo al iniciar la aplicacion.
	SeedOnStart bool
	// PartnerUsersAppDSN apunta a la base app/mysql del port del controller PHP.
	PartnerUsersAppDSN string
	// PartnerUsersProdDSN apunta a la base db_prod usada por tablas ISP_*_subscribers.
	PartnerUsersProdDSN string
	// PartnerAPIUser identifica al partner tecnico para consumir la API externa.
	PartnerAPIUser string
	// PartnerAPIPass es la credencial del partner tecnico.
	PartnerAPIPass string
	// PartnerAPIBaseURL es la raiz de la API externa del partner.
	PartnerAPIBaseURL string
	// PartnerLoginBaseURL es la base para construir URLs finales de login.
	PartnerLoginBaseURL string
	// PartnerAPITimeout controla el timeout del cliente HTTP externo.
	PartnerAPITimeout time.Duration
	// PartnerAPISkipTLSVerify replica el verify=false del proyecto PHP original.
	PartnerAPISkipTLSVerify bool
}

// Load construye la configuracion completa usando variables de entorno con
// valores por defecto razonables para desarrollo local.
func Load() Config {
	loadDotEnv(".env")

	return Config{
		AppName:          getEnv("APP_NAME", "tienda-go"),
		Env:              getEnv("APP_ENV", "development"),
		Address:          resolveAddress(),
		ReadTimeout:      getEnvDuration("APP_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:     getEnvDuration("APP_WRITE_TIMEOUT", 10*time.Second),
		ShutdownWait:     getEnvDuration("APP_SHUTDOWN_WAIT", 10*time.Second),
		TokenTTL:         getEnvDuration("AUTH_TOKEN_TTL", 12*time.Hour),
		LowStockLimit:    getEnvInt64("LOW_STOCK_LIMIT", 5),
		DefaultCurrency:  getEnv("APP_CURRENCY", "COP"),
		StoreDriver:      getEnv("STORE_DRIVER", "memory"),
		MySQLDSN:         getEnv("MYSQL_DSN", ""),
		MySQLAutoMigrate: getEnvBool("MYSQL_AUTO_MIGRATE", true),
		SeedOnStart:      getEnvBool("APP_SEED_ON_START", true),
		PartnerUsersAppDSN: firstNonEmpty(
			getEnv("PARTNER_USERS_APP_DSN", ""),
			buildMySQLDSNFromPrefix("DB"),
		),
		PartnerUsersProdDSN: firstNonEmpty(
			getEnv("PARTNER_USERS_PROD_DSN", ""),
			buildMySQLDSNFromPrefix("PROD_DB"),
		),
		PartnerAPIUser:          getEnv("PARTNER_API_USER", ""),
		PartnerAPIPass:          getEnv("PARTNER_API_PASS", ""),
		PartnerAPIBaseURL:       trimQuoted(getEnv("PARTNER_API_BASE_URL", "")),
		PartnerLoginBaseURL:     trimQuoted(getEnv("PARTNER_LOGIN_BASE_URL", "https://nuplin.com")),
		PartnerAPITimeout:       getEnvDuration("PARTNER_API_TIMEOUT", 20*time.Second),
		PartnerAPISkipTLSVerify: getEnvBool("PARTNER_API_SKIP_TLS_VERIFY", true),
	}
}

// getEnv devuelve el valor configurado o un fallback si no existe.
func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}

	return fallback
}

// getEnvDuration parsea duraciones como "5s" o "12h" y usa fallback si fallan.
func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

// getEnvInt64 parsea enteros para configuraciones numericas simples.
func getEnvInt64(key string, fallback int64) int64 {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}

	return parsed
}

// getEnvBool interpreta banderas simples de configuracion.
func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func buildMySQLDSNFromPrefix(prefix string) string {
	host := getEnv(prefix+"_HOST", "")
	port := getEnv(prefix+"_PORT", "")
	database := getEnv(prefix+"_DATABASE", "")
	username := getEnv(prefix+"_USERNAME", "")
	password := getEnv(prefix+"_PASSWORD", "")

	if host == "" || port == "" || database == "" || username == "" {
		return ""
	}

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true",
		username,
		password,
		host,
		port,
		database,
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func trimQuoted(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

func resolveAddress() string {
	if value := strings.TrimSpace(os.Getenv("APP_ADDR")); value != "" {
		return value
	}

	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}

	return ":8080"
}
