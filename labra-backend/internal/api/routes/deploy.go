package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Deploy(s *fuego.Server) {
	fuego.PostStd(s, "/v1/apps/{id}/deploy", withAuth(handlers.CreateDeployHandler))
	fuego.GetStd(s, "/v1/apps/{id}/deploys", withAuth(handlers.GetAppDeploysHandler))
	fuego.GetStd(s, "/v1/deploys/{id}", withAuth(handlers.GetDeployHandler))
	fuego.GetStd(s, "/v1/deploys/{id}/logs", withAuth(handlers.GetDeployLogsHandler))
}
