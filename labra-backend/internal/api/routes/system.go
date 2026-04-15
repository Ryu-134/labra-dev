package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func SystemRoutes(s *fuego.Server) {
	fuego.GetStd(s, "/v1/system/services", withAuth(handlers.GetSystemServicesHandler))
}
