package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Webhooks(s *fuego.Server) {
	fuego.PostStd(s, "/v1/webhooks/github", handlers.GitHubWebhookHandler)
}
