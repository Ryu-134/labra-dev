package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Apps(s *fuego.Server) {
	fuego.PostStd(s, "/v1/apps", handlers.CreateAppHandler)
	fuego.GetStd(s, "/v1/apps", handlers.ListAppsHandler)
	fuego.GetStd(s, "/v1/apps/{id}", handlers.GetAppHandler)
	fuego.PatchStd(s, "/v1/apps/{id}", handlers.PatchAppHandler)
}
