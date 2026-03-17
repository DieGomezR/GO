package partnerusers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Service agrupa la orquestacion del controller PHP ya portada a Go.
// Las dependencias quedan como interfaces para que el caso de uso no conozca
// detalles concretos de SQL, framework o cliente HTTP.
type Service struct {
	users               UserRepository
	isps                ISPRepository
	subscribers         SubscriberRepository
	partnerAPI          PartnerAPIClient
	audits              EarlyDeactivationAuditRepository
	scheduler           Scheduler
	uploads             UploadStorage
	packages            PackageCatalog
	earlyPolicy         EarlyDeactivationPolicy
	vigenciaPolicy      VigenciaPolicyEngine
	partnerLoginBaseURL string
}

// NewService construye el servicio principal equivalente al controller legacy.
func NewService(
	users UserRepository,
	isps ISPRepository,
	subscribers SubscriberRepository,
	partnerAPI PartnerAPIClient,
	audits EarlyDeactivationAuditRepository,
	scheduler Scheduler,
	uploads UploadStorage,
	packages PackageCatalog,
	earlyPolicy EarlyDeactivationPolicy,
	vigenciaPolicy VigenciaPolicyEngine,
	partnerLoginBaseURL string,
) *Service {
	return &Service{
		users:               users,
		isps:                isps,
		subscribers:         subscribers,
		partnerAPI:          partnerAPI,
		audits:              audits,
		scheduler:           scheduler,
		uploads:             uploads,
		packages:            packages,
		earlyPolicy:         earlyPolicy,
		vigenciaPolicy:      vigenciaPolicy,
		partnerLoginBaseURL: strings.TrimRight(partnerLoginBaseURL, "/"),
	}
}

// basePartner replica el helper privado del controller PHP:
// quita el sufijo numerico final y el prefijo "qa" del partner.
func (s *Service) basePartner(partnerID string) string {
	partner := strings.ToLower(strings.TrimSpace(partnerID))

	if idx := strings.LastIndex(partner, "_"); idx >= 0 {
		last := partner[idx+1:]
		if isDigits(last) {
			partner = partner[:idx]
		}
	}

	if strings.HasPrefix(partner, "qa") {
		partner = strings.TrimPrefix(partner, "qa")
	}

	return partner
}

// normalize deja un string comparable usando solo [a-z0-9].
func (s *Service) normalize(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') {
			builder.WriteRune(char)
		}
	}

	return builder.String()
}

// normalizeCountryCode fuerza el conjunto soportado por el negocio.
func (s *Service) normalizeCountryCode(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "co", "pe", "ec":
		return normalized
	default:
		return ""
	}
}

// mapTerritoryToCountryCode traduce nombres de territorio a codigos usados por el sistema.
func (s *Service) mapTerritoryToCountryCode(territory string) string {
	normalized := strings.ToLower(strings.TrimSpace(territory))
	switch {
	case normalized == "":
		return ""
	case strings.Contains(normalized, "colombia") || normalized == "co":
		return "co"
	case strings.Contains(normalized, "peru") || normalized == "pe":
		return "pe"
	case strings.Contains(normalized, "ecuador") || normalized == "ec":
		return "ec"
	default:
		return ""
	}
}

// resolveCountryForPartnerID porta la logica de busqueda de pais del controller.
func (s *Service) resolveCountryForPartnerID(ctx context.Context, partnerID string, authUser AuthUser) string {
	partnerID = strings.TrimSpace(partnerID)
	basePartner := ""
	if partnerID != "" {
		basePartner = s.basePartner(partnerID)
	}

	var isp *ISP
	if partnerID != "" && s.isps != nil {
		isp, _ = s.isps.FindByPartnerID(ctx, partnerID)
	}
	if isp == nil && basePartner != "" && s.isps != nil {
		isp, _ = s.isps.FindByPartnerID(ctx, basePartner)
	}

	if isp != nil {
		if country := s.normalizeCountryCode(isp.Country); country != "" {
			return country
		}
		if country := s.mapTerritoryToCountryCode(isp.Territory); country != "" {
			return country
		}
	}

	return s.normalizeCountryCode(authUser.Country)
}

// resolveCountryForUserID replica Users -> partnerId -> resolveCountryForPartnerID.
func (s *Service) resolveCountryForUserID(ctx context.Context, userID int64, authUser AuthUser) string {
	if s.users == nil {
		return ""
	}
	partnerID, err := s.users.FindPartnerIDByID(ctx, userID)
	if err != nil {
		return ""
	}
	return s.resolveCountryForPartnerID(ctx, partnerID, authUser)
}

// buildEarlyDeactivationSignal replica el helper que antes delegaba a la policy.
func (s *Service) buildEarlyDeactivationSignal(ctx context.Context, userID int64, authUser AuthUser) EarlyDeactivationSignal {
	country := s.resolveCountryForUserID(ctx, userID, authUser)
	if s.earlyPolicy == nil {
		return EarlyDeactivationSignal{}
	}
	return s.earlyPolicy.Evaluate(country, strings.ToLower(strings.TrimSpace(authUser.Role)))
}

// auditEarlyDeactivationAttempt traslada la auditoria a un repositorio dedicado.
func (s *Service) auditEarlyDeactivationAttempt(
	ctx context.Context,
	targetUserID int64,
	targetPartnerID string,
	country string,
	signal EarlyDeactivationSignal,
	allowed bool,
	rejectionReason string,
	authUser AuthUser,
) {
	if s.audits == nil {
		return
	}

	_ = s.audits.Insert(ctx, EarlyDeactivationAuditEntry{
		TargetUserID:     targetUserID,
		TargetPartnerID:  targetPartnerID,
		ActorID:          authUser.ID,
		ActorEmail:       authUser.ActorEmail(),
		ActorRole:        strings.ToLower(strings.TrimSpace(authUser.Role)),
		Country:          strings.ToLower(strings.TrimSpace(country)),
		Timezone:         signal.Timezone,
		Reason:           "portfolio_cutoff",
		Forced:           true,
		Allowed:          allowed,
		RejectionReason:  rejectionReason,
		WindowOpen:       signal.WindowOpen,
		AttemptedAtUTC:   time.Now().UTC(),
		AttemptedAtLocal: signal.NowLocal,
		Metadata: map[string]any{
			"canForceDeactivateEarly": signal.CanForceDeactivateEarly,
			"countryAllowed":          signal.CountryAllowed,
			"roleAllowed":             signal.RoleAllowed,
			"windowDays":              signal.WindowDays,
		},
	})
}

// pickTableForPartner porta el algoritmo de resolucion exacta/directa/fuzzy del nombre de tabla.
func (s *Service) pickTableForPartner(ctx context.Context, partnerID string) (string, []string, error) {
	attempted := make([]string, 0, 8)
	if s.subscribers == nil {
		return "", attempted, nil
	}

	base := s.basePartner(partnerID)
	firstSegment := base
	if idx := strings.Index(base, "_"); idx >= 0 {
		firstSegment = base[:idx]
	}

	exactCandidates := []string{
		fmt.Sprintf("ISP_%s_subscribers", base),
		fmt.Sprintf("isp_%s_subscribers", base),
	}
	for _, candidate := range exactCandidates {
		attempted = append(attempted, candidate)
		exists, err := s.subscribers.ExactTableExists(ctx, candidate)
		if err != nil {
			return "", attempted, err
		}
		if exists {
			return candidate, attempted, nil
		}
	}

	directCandidates := uniqueStrings([]string{
		fmt.Sprintf("ISP_%s_subscribers", base),
		fmt.Sprintf("isp_%s_subscribers", base),
		fmt.Sprintf("ISP_%s_subscribers", firstSegment),
		fmt.Sprintf("isp_%s_subscribers", firstSegment),
	})
	for _, candidate := range directCandidates {
		attempted = append(attempted, candidate)
		name, err := s.subscribers.FindCaseInsensitiveTable(ctx, candidate)
		if err != nil {
			return "", attempted, err
		}
		if name != "" {
			return name, attempted, nil
		}
	}

	target := s.normalize(base)
	rows, err := s.subscribers.ListSubscriberTables(ctx)
	if err != nil {
		return "", attempted, err
	}

	best := ""
	bestScore := -1
	for _, table := range rows {
		attempted = append(attempted, table)
		middle, ok := extractSubscriberTableMiddle(table)
		if !ok {
			continue
		}

		middle = s.normalize(middle)
		score := 0
		switch {
		case middle == target:
			score = 1000
		case strings.Contains(middle, target) || strings.Contains(target, middle):
			score = 100
		}
		if score > bestScore {
			best = table
			bestScore = score
		}
	}

	return best, attempted, nil
}

// matchingSubscriberTablesForBase devuelve todas las tablas cuyo segmento medio
// coincide exactamente con el partner base normalizado.
func (s *Service) matchingSubscriberTablesForBase(ctx context.Context, partner string) ([]string, error) {
	if s.subscribers == nil {
		return nil, nil
	}

	target := s.normalize(s.basePartner(partner))
	if target == "" {
		return nil, nil
	}

	tables, err := s.subscribers.ListSubscriberTables(ctx)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, 4)
	for _, table := range tables {
		middle, ok := extractSubscriberTableMiddle(table)
		if !ok {
			continue
		}
		if s.normalize(middle) == target {
			matches = append(matches, table)
		}
	}

	return uniqueStrings(matches), nil
}

// computeNextPartnerIDFor traslada la logica del controller que calculaba el siguiente sufijo.
func (s *Service) computeNextPartnerIDFor(ctx context.Context, referencePartnerID string) (string, string, []string, error) {
	base := s.basePartner(referencePartnerID)
	table, attempted, err := s.pickTableForPartner(ctx, referencePartnerID)
	if err != nil {
		return "", "", attempted, err
	}
	if table == "" || s.users == nil || s.subscribers == nil {
		return "", table, attempted, nil
	}

	maxUsers, err := s.users.MaxPartnerSuffix(ctx, base)
	if err != nil {
		return "", "", attempted, err
	}
	maxWP, err := s.subscribers.MaxPartnerSuffixOnTable(ctx, table, base)
	if err != nil {
		return "", "", attempted, err
	}

	next := maxUsers
	if maxWP > next {
		next = maxWP
	}
	next++

	return fmt.Sprintf("%s_%d", base, next), table, attempted, nil
}

// parseAPIError conserva el mapeo de mensajes especifico del controller legacy.
func (s *Service) parseAPIError(errCode string) string {
	switch errCode {
	case "unknown user":
		return "El usuario especificado no existe."
	case "invalid token":
		return "El token de autenticacion de la API no es valido."
	default:
		if errCode == "" {
			return "Error desconocido de la API externa."
		}
		return errCode
	}
}

// tryReadFromCandidates conserva el helper legado que intentaba leer columnas de varias tablas.
func (s *Service) tryReadFromCandidates(
	ctx context.Context,
	reader CandidateTableReader,
	candidates []string,
	partnerID string,
	columns []string,
) (string, map[string]any, []string, string, error) {
	attempted := make([]string, 0, len(candidates))
	permissionErr := ""

	for _, candidate := range uniqueStrings(candidates) {
		attempted = append(attempted, candidate)
		row, err := reader.ReadPartnerColumns(ctx, candidate, partnerID, columns)
		if err == nil {
			return candidate, row, attempted, "", nil
		}
		if reader.IsTableNotFound(err) {
			continue
		}
		if reader.IsPermissionError(err) {
			permissionErr = err.Error()
			continue
		}
		return "", nil, attempted, permissionErr, err
	}

	return "", nil, attempted, permissionErr, nil
}

// resolveTargetPartnerID porta la heuristica de update:
// partnerId directo, Users.id, email o partnerId exacto sin underscore.
func (s *Service) resolveTargetPartnerID(ctx context.Context, raw string) (string, error) {
	if s.users == nil {
		return "", ErrNotFound
	}

	value := strings.TrimSpace(raw)
	switch {
	case strings.Contains(value, "_"):
		return value, nil
	case isDigits(value):
		userID, err := parseInt64(value)
		if err != nil {
			return "", err
		}
		return s.users.FindPartnerIDByID(ctx, userID)
	case strings.Contains(value, "@"):
		return s.users.FindPartnerIDByEmail(ctx, strings.ToLower(value))
	default:
		user, err := s.users.FindByPartnerID(ctx, value)
		if err != nil || user == nil {
			return "", ErrNotFound
		}
		return value, nil
	}
}

func ensureAuthUser(authUser AuthUser) error {
	if strings.TrimSpace(authUser.PartnerID) == "" {
		return fmt.Errorf("%w: partnerid not available", ErrUnauthorized)
	}
	return nil
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func extractSubscriberTableMiddle(table string) (string, bool) {
	switch {
	case strings.HasPrefix(table, "ISP_") && strings.HasSuffix(table, "_subscribers"):
		return strings.TrimSuffix(strings.TrimPrefix(table, "ISP_"), "_subscribers"), true
	case strings.HasPrefix(table, "isp_") && strings.HasSuffix(table, "_subscribers"):
		return strings.TrimSuffix(strings.TrimPrefix(table, "isp_"), "_subscribers"), true
	default:
		return "", false
	}
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func parseInt64(value string) (int64, error) {
	var result int64
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("%w: %q is not numeric", ErrValidation, value)
		}
		result = result*10 + int64(char-'0')
	}
	return result, nil
}

func copyTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func timePtr(value time.Time) *time.Time {
	return &value
}
