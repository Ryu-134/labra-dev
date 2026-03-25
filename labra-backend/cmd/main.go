package main

import (
	"fmt"

	"labra-backend/internal/api/routes"

	"github.com/lpernett/godotenv"
	"github.com/go-fuego/fuego" 	

)

const PORT = "8080"
const HOST = "localhost"

func main() {
	err := godotenv.Load("./../.env")
	if err != nil {
		fmt.Println(err)
	}

	listenOn := HOST + ":" + PORT

	s := fuego.NewServer(
		fuego.WithAddr(listenOn),
	)

	routes.HealthRoute(s)
	routes.Oauth(s)

	// TODO: probably switch this to TLS


	fmt.Println("Server starting on :", listenOn)
	s.Run()	
}
