package partnerusers

import (
	"io"
	"time"
)

// AuthUser representa el usuario autenticado que llega al caso de uso.
// Es el equivalente a $request->user() en Laravel.
type AuthUser struct {
	ID        int64
	PartnerID string
	Email     string
	UserEmail string
	Role      string
	Country   string
}

// ActorEmail devuelve el mejor identificador de email disponible para logs y auditoria.
func (u AuthUser) ActorEmail() string {
	if u.UserEmail != "" {
		return u.UserEmail
	}
	return u.Email
}

// RegisterRequest modela el payload del metodo register del controller PHP.
type RegisterRequest struct {
	Email                 string     `json:"email"`
	Password              string     `json:"password"`
	Name                  string     `json:"name"`
	Package               string     `json:"package"`
	AdditionalChannels    []string   `json:"additional_channels"`
	PackageActivationMode string     `json:"package_activation_mode"`
	DefinedPackageDate    *time.Time `json:"defined_package_date"`
	Department            string     `json:"department"`
	City                  string     `json:"city"`
}

// UpdateRequest modela el payload del metodo update.
type UpdateRequest struct {
	Target string `json:"target"`
	Name   string `json:"name"`

	Package               string     `json:"package"`
	DefinedPackageDate    *time.Time `json:"defined_package_date"`
	PackageActivationMode string     `json:"package_activation_mode"`

	Password string `json:"password"`

	AdditionalChannels     []ChannelActivationRequest `json:"additional_channels"`
	ChannelsActivationMode string                     `json:"channels_activation_mode"`
	DefinedChannelsDate    *time.Time                 `json:"defined_channels_date"`
}

// ChannelActivationRequest permite representar tanto el arreglo simple como la version extendida.
type ChannelActivationRequest struct {
	Name string     `json:"name"`
	Date *time.Time `json:"date"`
}

// ActivatePackageRequest corresponde al metodo activate.
type ActivatePackageRequest struct {
	UserID         int64      `json:"user_id"`
	Package        string     `json:"package"`
	DefinedEndDate *time.Time `json:"defined_end_date"`
}

// RemoveRequest corresponde al metodo remove.
type RemoveRequest struct {
	UserID int64 `json:"user_id"`
}

// ActivateChannelsRequest corresponde al metodo activateChannels.
type ActivateChannelsRequest struct {
	PartnerID string     `json:"partner_id"`
	Channel   string     `json:"channel"`
	To        *time.Time `json:"to"`
}

// UserDetailsRequest corresponde al metodo getUserDetails.
type UserDetailsRequest struct {
	PartnerID string `json:"partner_id"`
}

// ListUsersByPartnerRequest lista usuarios a partir de un partner base.
type ListUsersByPartnerRequest struct {
	Partner string `json:"partner"`
	Limit   int    `json:"limit"`
}

// EarlyDeactivationStatusRequest corresponde al metodo getEarlyDeactivationStatus.
type EarlyDeactivationStatusRequest struct {
	Country string `json:"country"`
}

// DeactivateRequest corresponde al metodo deactivate.
type DeactivateRequest struct {
	UserID     int64 `json:"user_id"`
	ForceEarly bool  `json:"force_early"`
}

// BulkOperationRequest cubre las operaciones masivas por lista de user IDs.
type BulkOperationRequest struct {
	UserIDs []int64 `json:"user_ids"`
}

// ChangePackageMultipleRequest corresponde al metodo changePackageMultiple.
type ChangePackageMultipleRequest struct {
	UserIDs        []int64    `json:"user_ids"`
	NewPackage     string     `json:"new_package"`
	PackageEndDate *time.Time `json:"package_end_date"`
}

// GenerateLoginTokenRequest corresponde al metodo generateUserLoginToken.
type GenerateLoginTokenRequest struct {
	UserID             int64 `json:"user_id"`
	RedirectToServices bool  `json:"redirect_to_services"`
}

// GenerateLoginTokenByEmailRequest corresponde al metodo generateUserLoginTokenEmail.
type GenerateLoginTokenByEmailRequest struct {
	Email string `json:"email"`
}

// LogoutAllDevicesRequest corresponde al metodo logoutAllDevices.
type LogoutAllDevicesRequest struct {
	UserID int64 `json:"user_id"`
}

// ReactivateDeletedUserRequest corresponde al metodo reactivateDeletedUser.
type ReactivateDeletedUserRequest struct {
	Email                 string     `json:"email"`
	Password              string     `json:"password"`
	Name                  string     `json:"name"`
	Package               string     `json:"package"`
	AdditionalChannels    []string   `json:"additional_channels"`
	PackageActivationMode string     `json:"package_activation_mode"`
	DefinedPackageDate    *time.Time `json:"defined_package_date"`
	Department            string     `json:"department"`
	City                  string     `json:"city"`
}

// QueueBulkUploadRequest modela la carga de archivo del metodo upload.
type QueueBulkUploadRequest struct {
	FileName string
	Content  io.Reader
}

// RegisterResult modela la respuesta exitosa del caso de uso register.
type RegisterResult struct {
	Message           string `json:"message"`
	AssignedPartnerID string `json:"assigned_partner_id"`
	UsedTable         string `json:"used_table"`
	ExternalID        string `json:"external_id"`
}

// UpdateResult modela la respuesta del caso de uso update.
type UpdateResult struct {
	Message   string              `json:"message"`
	PartnerID string              `json:"partner_id"`
	UsedTable string              `json:"used_table"`
	Channels  []ChannelActivation `json:"channels"`
}

// ActivatePackageResult modela la respuesta del caso de uso activate.
type ActivatePackageResult struct {
	Message string `json:"message"`
}

// LocalUpdateStatus resume el efecto local en la tabla de suscriptores.
type LocalUpdateStatus struct {
	OK       bool   `json:"ok"`
	Affected int64  `json:"affected"`
	Table    string `json:"table"`
	By       string `json:"by"`
	Error    string `json:"error,omitempty"`
}

// OperationStatus representa el resultado de una operacion externa.
type OperationStatus struct {
	OK     bool   `json:"ok"`
	Status int    `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// RemoveResult modela la respuesta del caso de uso remove.
type RemoveResult struct {
	Message  string            `json:"message"`
	External OperationStatus   `json:"external"`
	Local    LocalUpdateStatus `json:"local"`
}

// DeactivateResult modela la respuesta del caso de uso deactivate.
type DeactivateResult struct {
	Message string     `json:"message"`
	UserID  int64      `json:"user_id,omitempty"`
	RunAt   *time.Time `json:"run_at,omitempty"`
}

// BulkResult devuelve el detalle por usuario en operaciones masivas.
type BulkResult struct {
	UserID  int64  `json:"user_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// GenerateLoginTokenResult modela la respuesta del token de login.
type GenerateLoginTokenResult struct {
	Message  string `json:"message"`
	Token    string `json:"token"`
	TokenURL string `json:"token_url"`
}

// QueueBulkUploadResult modela la respuesta del upload masivo.
type QueueBulkUploadResult struct {
	Message string `json:"message"`
	Path    string `json:"path"`
}

// UserDevice representa un dispositivo del usuario en la respuesta.
type UserDevice struct {
	Type           string     `json:"type"`
	Name           string     `json:"name"`
	LastConnection *time.Time `json:"last_connection,omitempty"`
	IP             string     `json:"ip,omitempty"`
	CreatedAt      *time.Time `json:"created_at,omitempty"`
}

// EarlyDeactivationSignal corresponde a la evaluacion de la policy de desactivacion temprana.
type EarlyDeactivationSignal struct {
	Country                 string     `json:"country"`
	Timezone                string     `json:"timezone"`
	WindowDays              []int      `json:"window_days"`
	WindowOpen              bool       `json:"window_open"`
	CanForceDeactivateEarly bool       `json:"can_force_deactivate_early"`
	RoleAllowed             bool       `json:"role_allowed"`
	CountryAllowed          bool       `json:"country_allowed"`
	Reason                  string     `json:"reason"`
	NowLocal                *time.Time `json:"now_local,omitempty"`
	NowUTC                  *time.Time `json:"now_utc,omitempty"`
}

// UserDetailsResult corresponde al payload final de getUserDetails.
type UserDetailsResult struct {
	FullName                string                  `json:"full_name"`
	ActivationDate          *time.Time              `json:"activation_date,omitempty"`
	ActiveDays              *int                    `json:"active_days,omitempty"`
	CurrentPackage          string                  `json:"current_package"`
	DeactivationDate        *time.Time              `json:"deactivation_date,omitempty"`
	Devices                 []UserDevice            `json:"devices"`
	CanForceDeactivateEarly bool                    `json:"can_force_deactivate_early"`
	ForceDeactivatePolicy   EarlyDeactivationSignal `json:"force_deactivate_policy"`
}

// PartnerUserSummary resume una fila encontrada para un partner base.
type PartnerUserSummary struct {
	Table        string `json:"table"`
	SubscriberID int64  `json:"subscriber_id,omitempty"`
	PartnerID    string `json:"partner_id"`
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status,omitempty"`
	Package      string `json:"package,omitempty"`
}

// ListUsersByPartnerResult devuelve todas las filas encontradas por partner base.
type ListUsersByPartnerResult struct {
	Partner string               `json:"partner"`
	Total   int                  `json:"total"`
	Users   []PartnerUserSummary `json:"users"`
}

// LoginTokenByEmailResult modela el redirect final del flujo por email.
type LoginTokenByEmailResult struct {
	RedirectURL string `json:"redirect_url"`
}

// ChannelActivation representa un canal activado y su fecha opcional.
type ChannelActivation struct {
	Name string     `json:"name"`
	Date *time.Time `json:"date,omitempty"`
}

// RegisterAPIResult representa la respuesta de la API externa al registrar usuario.
type RegisterAPIResult struct {
	ExternalID      string
	ReadablePackage string
	RawResponse     any
}

// ExternalService representa bundles o servicios individuales reportados por la API externa.
type ExternalService struct {
	Type string
	Code string
	From *time.Time
	To   *time.Time
}

// ExternalDevice representa un dispositivo reportado por la API externa.
type ExternalDevice struct {
	ID        string
	Type      string
	Title     string
	LastLogin *time.Time
	LastIP    string
	CreatedAt *time.Time
	Enabled   bool
}

// ExternalUserDetails modela una respuesta normalizada de get-user.
type ExternalUserDetails struct {
	FullName string
	Services []ExternalService
	Devices  []ExternalDevice
}

// STBDevice representa dispositivos obtenidos por get-stbs o get-smart-tvs.
type STBDevice struct {
	DeviceID string
}

// SubscriberRecord representa la fila relevante de una tabla ISP_*_subscribers.
type SubscriberRecord struct {
	SubscriberID       int64
	PartnerID          string
	Email              string
	Name               string
	Status             string
	Package            string
	StartDate          *time.Time
	EndDate            *time.Time
	AdditionalChannels []ChannelActivation
	RemovedAt          *time.Time
	LastUpdatedAt      *time.Time
}

// SubscriberUpdate modela los cambios parciales sobre la tabla de suscriptores.
type SubscriberUpdate struct {
	PartnerID          *string
	SubscriberID       *int64
	Status             *string
	Name               *string
	Package            *string
	StartDate          *time.Time
	EndDate            *time.Time
	Department         *string
	City               *string
	AdditionalChannels []ChannelActivation
	RemovedAt          *time.Time
	LastUpdatedAt      *time.Time
}

// ManagedUser representa la fila local de Users usada por varios casos de uso.
type ManagedUser struct {
	ID        int64
	PartnerID string
	Email     string
}

// ISP representa la informacion del partner usada para resolver pais.
type ISP struct {
	PartnerID string
	Country   string
	Territory string
}

// EarlyDeactivationAuditEntry representa la fila que el controller PHP insertaba
// en early_deactivation_audits.
type EarlyDeactivationAuditEntry struct {
	TargetUserID     int64
	TargetPartnerID  string
	ActorID          int64
	ActorEmail       string
	ActorRole        string
	Country          string
	Timezone         string
	Reason           string
	Forced           bool
	Allowed          bool
	RejectionReason  string
	WindowOpen       bool
	AttemptedAtUTC   time.Time
	AttemptedAtLocal *time.Time
	Metadata         map[string]any
}

// PolicyValidationResult modela el resultado de una policy de vigencia.
type PolicyValidationResult struct {
	OK          bool
	Error       string
	MinimumDate *time.Time
}
