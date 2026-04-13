package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"labra-backend/internal/api/handlers"
	"labra-backend/internal/api/routes"
	"labra-backend/internal/api/services"

	"github.com/go-fuego/fuego"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lpernett/godotenv"
)

const (
	PORT = "8080"
	HOST = "localhost"
)

var db_url string
var github_webhook_secret string

func init() {
	err := godotenv.Load("./../.env")
	if err != nil {
		fmt.Println(err)
	}

	gh_client := os.Getenv("GH_CLIENT_ID")
	gh_secret := os.Getenv("GH_CLIENT_SECRET")
	db_url = os.Getenv("DB_URL")
	github_webhook_secret = os.Getenv("GITHUB_WEBHOOK_SECRET")

	services.InitOauth(gh_client, gh_secret)
}

func main() {
	listenOn := HOST + ":" + PORT

	s := fuego.NewServer(
		fuego.WithAddr(listenOn),
	)

	// -- setup db --

	db, err := sql.Open("sqlite3", db_url)
	if err != nil {
		log.Fatalln(err)
	}
	defer db.Close()

	driver, err := sqlite3.WithInstance(db, &sqlite3.Config{})
	if err != nil {
		log.Fatalln(err)
	}

	// maybe change this path later
	m, err := migrate.NewWithDatabaseInstance(
		"file://../sql/migrations",
		"sqlite3",
		driver,
	)

	if err != nil {
		log.Fatalln(err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalln(err)
	}

	fmt.Println("DB CONNECTED")
	handlers.InitAppStore(db)
	handlers.InitWebhook(github_webhook_secret)
	routes.HealthRoute(s)
	routes.Oauth(s)
	routes.Apps(s)
	routes.Deploy(s)
	routes.Webhooks(s)

	// TODO: probably switch this to TLS

	fmt.Println("Server starting on :", listenOn)
	s.Run()
}
