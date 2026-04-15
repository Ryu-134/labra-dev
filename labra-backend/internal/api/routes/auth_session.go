package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func AuthSessionRoutes(s *fuego.Server) {
	fuego.PostStd(s, "/v1/auth/session", handlers.PostAuthSessionHandler)
	fuego.GetStd(s, "/v1/profile", withAuth(handlers.GetProfileHandler))
	fuego.PostStd(s, "/v1/auth/logout", withAuth(handlers.PostLogoutHandler))
}
