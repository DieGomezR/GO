package api

import (
	"net/http"
	"strconv"
	"strings"

	"tienda-go/internal/partnerusers"
)

// PartnerUsersCurrentUser adapta el usuario autenticado de la API principal al
// shape esperado por el port del controller PHP. Para Postman se completa el
// contexto partner con headers X-Partner-*.
func (a *API) PartnerUsersCurrentUser(r *http.Request) partnerusers.AuthUser {
	storeUser := currentUser(r)
	actorID, _ := strconv.ParseInt(strings.TrimSpace(r.Header.Get("X-Partner-Actor-ID")), 10, 64)

	email := firstNonEmptyHeader(
		r,
		"X-Partner-Email",
		"X-User-Email",
	)
	if email == "" {
		email = storeUser.Email
	}

	role := firstNonEmptyHeader(r, "X-Partner-Role")
	if role == "" {
		role = string(storeUser.Role)
	}

	country := firstNonEmptyHeader(r, "X-Partner-Country")
	if country == "" {
		country = "co"
	}

	return partnerusers.AuthUser{
		ID:        actorID,
		PartnerID: firstNonEmptyHeader(r, "X-Partner-ID"),
		Email:     email,
		UserEmail: email,
		Role:      role,
		Country:   country,
	}
}

func firstNonEmptyHeader(r *http.Request, keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(r.Header.Get(key))
		if value != "" {
			return value
		}
	}
	return ""
}
