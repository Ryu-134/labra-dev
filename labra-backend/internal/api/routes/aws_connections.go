package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func AWSConnections(s *fuego.Server) {
	fuego.PostStd(s, "/v1/aws-connections", handlers.UpsertAWSConnectionHandler)
	fuego.GetStd(s, "/v1/aws-connections", handlers.ListAWSConnectionsHandler)
}
