package main

import (
	"log"
	"os"
	"trading_bot/internal/clients/postgres"

	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

const (
	migrationsDir = "./migrations/postgres"
	cmdUp         = "up"
	cmdDown       = "down"
	cmdStatus     = "status"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("migration action is required: %s/%s/%s", cmdUp, cmdDown, cmdStatus)
	}
	command := os.Args[1]

	db, err := postgres.NewClient()
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
	defer db.GetDB().Close()

	goose.SetBaseFS(nil)

	if command == cmdUp {
		if err := goose.Up(db.GetDB(), migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else if command == cmdDown {
		if err := goose.Down(db.GetDB(), migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else if command == cmdStatus {
		if err := goose.Status(db.GetDB(), migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else {
		log.Fatalf("Wrong comand send: %s. Required: %s/%s/%s", command, cmdUp, cmdDown, cmdStatus)
	}

	log.Println("Migrations command successfully done")
}
