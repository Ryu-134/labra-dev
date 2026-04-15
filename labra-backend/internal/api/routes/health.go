package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func HealthRoute(s *fuego.Server) {
	fuego.GetStd(s, "/health", handlers.HandleHealth)
	fuego.GetStd(s, "/ready", handlers.HandleReadiness)
}
