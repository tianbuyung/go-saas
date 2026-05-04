package main

import (
	"errors"
	"log"
	"os"
	"strconv"

	"saas/internal/config"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	cfg := config.Load()

	m, err := migrate.New(
		"file://internal/db/migrations",
		cfg.DBUrl,
	)
	if err != nil {
		log.Fatal(err)
	}

	if len(os.Args) < 2 {
		log.Fatal("expected command: up | down | force | drop")
	}

	cmd := os.Args[1]

	log.Printf("running migration command: %s", cmd)

	if cfg.Env == "production" && (cmd == "down" || cmd == "drop") {
		log.Fatal("down/drop not allowed in production")
	}

	switch cmd {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal(err)
		}

	case "down":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			log.Fatal(err)
		}

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("force requires version")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatal("invalid version number")
		}
		if err := m.Force(v); err != nil {
			log.Fatal(err)
		}

	case "drop":
		if err := m.Drop(); err != nil {
			log.Fatal(err)
		}

	default:
		log.Fatal("unknown command")
	}

	version, dirty, err := m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		log.Fatal(err)
	}

	log.Printf("current version: %d dirty: %v", version, dirty)
	log.Println("migration done:", cmd)
}
