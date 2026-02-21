package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/LittleAksMax/bids-util/env"
	"github.com/joho/godotenv"

	"github.com/LittleAksMax/bids-user-service/api"
	"github.com/LittleAksMax/bids-user-service/config"
	"github.com/LittleAksMax/bids-user-service/db"
)

const (
	ModeDevelopment = "development"
	ModeProduction  = "production"
)

func main() {
	// Load development override file BEFORE config parsing if MODE indicates development.
	mode := env.GetStrFromEnv("MODE")
	if mode != ModeDevelopment && mode != ModeProduction {
		log.Fatalf("invalid environment variable MODE: %s", mode)
	}
	if mode == ModeDevelopment {
		if err := godotenv.Load(".env.Dev"); err != nil {
			log.Fatalf("Failed to load .env.Dev: %v", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	dsn := cfg.DSN()
	pool, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	defer func() {
		if err := pool.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	}()

	// Migrate automatically if in development mode
	if mode == ModeDevelopment {
		if err := db.Migrate(dsn, "migrations"); err != nil {
			log.Fatalf("migration error: %v", err)
		}
	}

	r := api.NewRouter(pool, cfg, mode == ModeProduction)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("starting server on %s (mode=%s)", addr, mode)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Printf("server stopped: %v", err)
		os.Exit(1)
	}
}
