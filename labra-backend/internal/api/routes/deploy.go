package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Deploy(s *fuego.Server) {
	fuego.PostStd(s, "/v1/apps/{id}/deploy", handlers.CreateDeployHandler)
	fuego.GetStd(s, "/v1/apps/{id}/deploys", handlers.GetAppDeploysHandler)
	fuego.GetStd(s, "/v1/deploys/{id}", handlers.GetDeployHandler)
	fuego.GetStd(s, "/v1/deploys/{id}/logs", handlers.GetDeployLogsHandler)
}
