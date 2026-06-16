package main

import (
	"log"

	"github.com/sharknado/backend/internal/config"
	"github.com/sharknado/backend/internal/events"
	"github.com/sharknado/backend/internal/library"
	"github.com/sharknado/backend/internal/server"
)

func main() {
	cfg := config.Load()

	broker := events.NewEventBroker()

	db, err := library.NewDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	srv := server.NewServer(cfg, db, broker)

	if err := srv.Start(); err != nil {
		log.Fatalf("server shutdown with error: %v", err)
	}
}
