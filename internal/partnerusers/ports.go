package partnerusers

import (
	"context"
	"io"
	"time"
)

// PartnerAPIClient encapsula las llamadas que en Laravel iban a config('services.partner_api') + Http::get/post.
type PartnerAPIClient interface {
	RegisterUser(ctx context.Context, req RegisterRequest, partnerID, country string) (RegisterAPIResult, error)
	ModifyUser(ctx context.Context, partnerID, name, password string) error
	ActivatePackage(ctx context.Context, partnerID, packageCode string, to *time.Time) error
	ActivateServices(ctx context.Context, partnerID, bundle string, to *time.Time) error
	RemoveUser(ctx context.Context, partnerID string, userID int64) (OperationStatus, error)
	DeactivateUser(ctx context.Context, partnerID string, all bool) (OperationStatus, error)
	GetUser(ctx context.Context, partnerID string, userID int64) (ExternalUserDetails, error)
	GetSTBs(ctx context.Context, partnerID string) ([]STBDevice, error)
	GetSmartTVs(ctx context.Context, partnerID string) ([]STBDevice, error)
	RemoveDevice(ctx context.Context, deviceID string) error
	GenerateLoginToken(ctx context.Context, userID int64, redirectToServices bool) (string, error)
}

// UserRepository encapsula la tabla Users usada por el controller PHP.
type UserRepository interface {
	FindByID(ctx context.Context, userID int64) (*ManagedUser, error)
	FindByPartnerID(ctx context.Context, partnerID string) (*ManagedUser, error)
	FindPartnerIDByID(ctx context.Context, userID int64) (string, error)
	FindPartnerIDByEmail(ctx context.Context, email string) (string, error)
	MaxPartnerSuffix(ctx context.Context, base string) (int, error)
	SaveRegisteredUser(ctx context.Context, req RegisterRequest, partnerID, readablePackage, country string, channels []ChannelActivation) error
}

// ISPRepository encapsula la tabla Isp y su configuracion para resolver pais.
type ISPRepository interface {
	FindByPartnerID(ctx context.Context, partnerID string) (*ISP, error)
}

// SubscriberRepository encapsula las tablas ISP_*_subscribers en la base db_prod.
type SubscriberRepository interface {
	ExactTableExists(ctx context.Context, tableName string) (bool, error)
	FindCaseInsensitiveTable(ctx context.Context, tableName string) (string, error)
	ListSubscriberTables(ctx context.Context) ([]string, error)
	MaxPartnerSuffixOnTable(ctx context.Context, tableName, base string) (int, error)
	FindByPartnerID(ctx context.Context, tableName, partnerID string) (*SubscriberRecord, error)
	FindBySubscriberID(ctx context.Context, tableName string, userID int64) (*SubscriberRecord, error)
	FindByEmailAndStates(ctx context.Context, tableName, email string, states []string) (*SubscriberRecord, error)
	ListByPartnerPrefix(ctx context.Context, tableName, partnerPrefix string, limit int) ([]SubscriberRecord, error)
	CreateMirroredUser(ctx context.Context, tableName string, req RegisterRequest, partnerID, readablePackage string, channels []ChannelActivation, externalID string, endDate *time.Time) error
	UpdateByPartnerID(ctx context.Context, tableName, partnerID string, update SubscriberUpdate) (int64, error)
	UpdateBySubscriberID(ctx context.Context, tableName string, userID int64, update SubscriberUpdate) (int64, error)
	UpdateByEmailAndStates(ctx context.Context, tableName, email string, states []string, update SubscriberUpdate) (int64, error)
}

// EarlyDeactivationAuditRepository encapsula la auditoria de intentos de desactivacion temprana.
type EarlyDeactivationAuditRepository interface {
	Insert(ctx context.Context, entry EarlyDeactivationAuditEntry) error
}

// Scheduler encapsula jobs diferidos y colas en segundo plano.
type Scheduler interface {
	ScheduleDeactivation(ctx context.Context, userID int64, actorEmail, clientIP string, runAt time.Time) error
	DispatchBulkUsers(ctx context.Context, path, actorEmail string) error
}

// UploadStorage encapsula el guardado temporal del archivo JSON del upload masivo.
type UploadStorage interface {
	Save(ctx context.Context, fileName string, content io.Reader) (string, error)
}

// PackageCatalog reemplaza a PackageHelper y concentra traducciones de paquete/canales.
type PackageCatalog interface {
	OriginalCode(readable string) (string, bool)
	ReadableName(codeOrName string) string
	ChannelsToBundles(channelNames []string) []string
}

// EarlyDeactivationPolicy reemplaza a EarlyDeactivationPolicyService.
type EarlyDeactivationPolicy interface {
	Evaluate(country, role string) EarlyDeactivationSignal
}

// VigenciaPolicyEngine reemplaza a VigenciaPolicy.
type VigenciaPolicyEngine interface {
	MinDefinedDate(start *time.Time) time.Time
	BarrierFromStart(start *time.Time) time.Time
	ValidateDefinedDateForSwitch(start *time.Time, proposed time.Time, isCurrentlyIndef bool) PolicyValidationResult
	RunAtForDeactivation(start, end *time.Time) *time.Time
}

// CandidateTableReader conserva el helper tryReadFromCandidates del controller PHP.
type CandidateTableReader interface {
	ReadPartnerColumns(ctx context.Context, tableName, partnerID string, columns []string) (map[string]any, error)
	IsTableNotFound(err error) bool
	IsPermissionError(err error) bool
}
