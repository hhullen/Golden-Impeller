package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"trading_bot/internal/supports"

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

	cfg, err := supports.GetEnvCfg()
	if err != nil {
		log.Fatalln(err)
	}

	db, err := sql.Open("postgres",
		fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg["DB_HOST"], cfg["DB_PORT"], cfg["DB_USER"], cfg["DB_PASSWORD"], cfg["DB_NAME"]))
	if err != nil {
		log.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	goose.SetBaseFS(nil)

	if command == cmdUp {
		if err := goose.Up(db, migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else if command == cmdDown {
		if err := goose.Down(db, migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else if command == cmdStatus {
		if err := goose.Status(db, migrationsDir); err != nil {
			log.Fatalf("failed to apply migrations: %v", err)
		}
	} else {
		log.Fatalf("Wrong comand send: %s. Required: %s/%s/%s", command, cmdUp, cmdDown, cmdStatus)
	}

	log.Println("Migrations command successfully done")
}
