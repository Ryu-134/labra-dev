package routes

import (
	"net/http"

	"labra-backend/internal/api/auth"
)

var (
	requireAuthMiddleware = func(next http.Handler) http.Handler { return next }
)

func InitAuthMiddleware(v auth.Validator) {
	if v == nil {
		requireAuthMiddleware = func(next http.Handler) http.Handler { return next }
		return
	}
	requireAuthMiddleware = auth.RequireAuth(v)
}

func withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requireAuthMiddleware(handler).ServeHTTP(w, r)
	}
}

func withAuthAndRoles(handler http.HandlerFunc, roles ...string) http.HandlerFunc {
	roleGuard := auth.RequireAnyRole(roles...)
	return func(w http.ResponseWriter, r *http.Request) {
		requireAuthMiddleware(roleGuard(handler)).ServeHTTP(w, r)
	}
}
