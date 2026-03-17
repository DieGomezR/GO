package partnerusers

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// DefaultPackageCatalog deja pasar los nombres tal como llegan mientras no
// tengamos el mapeo completo que hoy vive en PackageHelper dentro del proyecto PHP.
type DefaultPackageCatalog struct{}

// OriginalCode devuelve el valor tal cual, siempre que no este vacio.
func (DefaultPackageCatalog) OriginalCode(readable string) (string, bool) {
	readable = strings.TrimSpace(readable)
	return readable, readable != ""
}

// ReadableName devuelve el codigo recibido sin alterar.
func (DefaultPackageCatalog) ReadableName(codeOrName string) string {
	return strings.TrimSpace(codeOrName)
}

// ChannelsToBundles reutiliza el nombre del canal como bundle minimo viable.
func (DefaultPackageCatalog) ChannelsToBundles(channelNames []string) []string {
	result := make([]string, 0, len(channelNames))
	for _, name := range channelNames {
		name = strings.TrimSpace(name)
		if name != "" {
			result = append(result, name)
		}
	}
	return result
}

// DefaultEarlyDeactivationPolicy replica una politica simple y explicita para
// no dejar el endpoint sin comportamiento mientras falta la policy real.
type DefaultEarlyDeactivationPolicy struct{}

// Evaluate calcula una señal utilizable por los handlers actuales.
func (DefaultEarlyDeactivationPolicy) Evaluate(country, role string) EarlyDeactivationSignal {
	nowUTC := time.Now().UTC()
	location, err := time.LoadLocation("America/Bogota")
	if err != nil {
		location = time.FixedZone("COT", -5*60*60)
	}
	nowLocal := nowUTC.In(location)

	country = strings.ToLower(strings.TrimSpace(country))
	role = strings.ToLower(strings.TrimSpace(role))

	countryAllowed := country == "co" || country == "pe" || country == "ec"
	roleAllowed := role == "admin" || role == "manager"
	windowOpen := nowLocal.Day() >= 1 && nowLocal.Day() <= 6
	canForce := countryAllowed && roleAllowed && windowOpen

	reason := "policy_denied"
	switch {
	case canForce:
		reason = "allowed"
	case !countryAllowed:
		reason = "country_not_allowed"
	case !roleAllowed:
		reason = "role_not_allowed"
	case !windowOpen:
		reason = "outside_window"
	}

	return EarlyDeactivationSignal{
		Country:                 country,
		Timezone:                location.String(),
		WindowDays:              []int{1, 6},
		WindowOpen:              windowOpen,
		CanForceDeactivateEarly: canForce,
		RoleAllowed:             roleAllowed,
		CountryAllowed:          countryAllowed,
		Reason:                  reason,
		NowLocal:                copyTime(&nowLocal),
		NowUTC:                  copyTime(&nowUTC),
	}
}

// DefaultVigenciaPolicy implementa una version compacta de la politica de vigencia.
type DefaultVigenciaPolicy struct{}

// MinDefinedDate calcula la barrera minima.
func (DefaultVigenciaPolicy) MinDefinedDate(start *time.Time) time.Time {
	if start == nil {
		return time.Now().UTC().AddDate(0, 1, 0)
	}
	return start.UTC().AddDate(0, 1, 0)
}

// BarrierFromStart reutiliza la misma barrera minima.
func (p DefaultVigenciaPolicy) BarrierFromStart(start *time.Time) time.Time {
	return p.MinDefinedDate(start)
}

// ValidateDefinedDateForSwitch valida una fecha propuesta.
func (p DefaultVigenciaPolicy) ValidateDefinedDateForSwitch(start *time.Time, proposed time.Time, _ bool) PolicyValidationResult {
	minimum := p.BarrierFromStart(start)
	if proposed.Before(minimum) {
		return PolicyValidationResult{
			OK:          false,
			Error:       "defined date is before the allowed minimum",
			MinimumDate: &minimum,
		}
	}
	return PolicyValidationResult{OK: true}
}

// RunAtForDeactivation devuelve fin existente o la barrera desde inicio.
func (p DefaultVigenciaPolicy) RunAtForDeactivation(start, end *time.Time) *time.Time {
	if end != nil {
		return copyTime(end)
	}
	if start == nil {
		return nil
	}
	runAt := p.BarrierFromStart(start)
	return &runAt
}

// ErrorScheduler deja claro que la cola no esta integrada aun.
type ErrorScheduler struct{}

// ScheduleDeactivation falla explicitamente cuando no existe un scheduler real.
func (ErrorScheduler) ScheduleDeactivation(context.Context, int64, string, string, time.Time) error {
	return fmt.Errorf("%w: scheduler is not configured", ErrConflict)
}

// DispatchBulkUsers falla explicitamente cuando no existe un scheduler real.
func (ErrorScheduler) DispatchBulkUsers(context.Context, string, string) error {
	return fmt.Errorf("%w: scheduler is not configured", ErrConflict)
}
