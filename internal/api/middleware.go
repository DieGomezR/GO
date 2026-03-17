// Package api adapta los casos de uso de negocio a HTTP/JSON.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"tienda-go/internal/domain"
)

// contextKey evita colisiones de claves en el contexto HTTP.
type contextKey string

const userContextKey contextKey = "auth-user"

// authenticated exige un Bearer token valido y guarda el usuario en el contexto.
func (a *API) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r.Header.Get("Authorization"))
		user, err := a.auth.Authenticate(token)
		if err != nil {
			writeError(w, err)
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// requireRoles combina autenticacion y autorizacion por rol.
func (a *API) requireRoles(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return a.authenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := currentUser(r)
			for _, role := range roles {
				if user.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}

			writeError(w, domain.ErrForbidden)
		}))
	}
}

// logging mide cada request y la escribe en slog.
func (a *API) logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		a.logger.Info("request completed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", recorder.status),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// recoverer evita que un panic tumbe todo el servidor.
func (a *API) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", slog.Any("error", recovered))
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error": "internal server error",
				})
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// currentUser extrae del contexto al usuario ya autenticado por middleware.
func currentUser(r *http.Request) domain.User {
	user, _ := r.Context().Value(userContextKey).(domain.User)
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

// statusRecorder captura el codigo de respuesta para logging.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// WriteHeader intercepta el status y luego delega al ResponseWriter real.
func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}
