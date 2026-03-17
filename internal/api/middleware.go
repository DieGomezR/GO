// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"tienda-go/internal/domain"
)

const userContextKey = "auth-user"

// authenticated exige un Bearer token valido y guarda el usuario en Fiber locals.
func (a *API) authenticated() fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c.Get("Authorization"))
		user, err := a.auth.Authenticate(token)
		if err != nil {
			return writeError(c, err)
		}

		c.Locals(userContextKey, user)
		return c.Next()
	}
}

// requireRoles combina autenticacion y autorizacion por rol.
func (a *API) requireRoles(roles ...domain.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := extractBearerToken(c.Get("Authorization"))
		user, err := a.auth.Authenticate(token)
		if err != nil {
			return writeError(c, err)
		}

		c.Locals(userContextKey, user)

		for _, role := range roles {
			if user.Role == role {
				return c.Next()
			}
		}

		return writeError(c, domain.ErrForbidden)
	}
}

// logging mide cada request y la escribe en slog.
func (a *API) logging() fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		defer func() {
			a.logger.Info("request completed",
				slog.String("method", c.Method()),
				slog.String("path", c.Path()),
				slog.Int("status", c.Response().StatusCode()),
				slog.Duration("duration", time.Since(start)),
			)
		}()

		return c.Next()
	}
}

// recoverer evita que un panic tumbe todo el servidor.
func (a *API) recoverer() fiber.Handler {
	return func(c *fiber.Ctx) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", slog.Any("error", recovered))
				err = writeJSON(c, fiber.StatusInternalServerError, map[string]any{
					"error": "internal server error",
				})
			}
		}()

		return c.Next()
	}
}

// currentUser extrae el usuario ya autenticado desde Fiber locals.
func currentUser(c *fiber.Ctx) domain.User {
	user, _ := c.Locals(userContextKey).(domain.User)
	return user
}

// extractBearerToken limpia el header Authorization y extrae el token.
func extractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}

	return strings.TrimSpace(header[7:])
}
