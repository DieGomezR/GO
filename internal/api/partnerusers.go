package api

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/domain"
	"tienda-go/internal/partnerusers"
)

// PartnerUsersCurrentUser adapta headers HTTP al shape esperado por el port del
// controller PHP usando primero el usuario autenticado resuelto por Fiber y
// luego permitiendo overrides explicitos via headers X-Partner-*.
func (a *API) PartnerUsersCurrentUser(c *fiber.Ctx) partnerusers.AuthUser {
	authUser, _ := c.Locals(userContextKey).(domain.User)
	actorID, _ := strconv.ParseInt(strings.TrimSpace(c.Get("X-Partner-Actor-ID")), 10, 64)
	if actorID == 0 {
		actorID, _ = strconv.ParseInt(strings.TrimSpace(authUser.ID), 10, 64)
	}

	email := firstNonEmptyValue(
		strings.TrimSpace(c.Get("X-Partner-Email")),
		strings.TrimSpace(authUser.Email),
	)

	role := firstNonEmptyValue(strings.TrimSpace(c.Get("X-Partner-Role")), string(authUser.Role))
	country := strings.TrimSpace(c.Get("X-Partner-Country"))
	if country == "" {
		country = "co"
	}

	return partnerusers.AuthUser{
		ID:        actorID,
		PartnerID: strings.TrimSpace(c.Get("X-Partner-ID")),
		Email:     email,
		UserEmail: email,
		Role:      role,
		Country:   country,
	}
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
