package routes

import (
	"labra-backend/internal/api/handlers"

	"github.com/go-fuego/fuego"
)

func Oauth(s *fuego.Server) {
	fuego.PostStd(s, "/login", handlers.LoginHandler)
	fuego.GetStd(s, "/callback", handlers.CallbackHandler)
}
