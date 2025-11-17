package main

import (
	"gollama/internal/config"
	"gollama/internal/handler"
	"gollama/internal/pool"
	"gollama/internal/server"
	"log"
)

func main() {
	// Initialize authentication with credentials from DB/auth.json
	err := handler.InitAuth("DB/auth.json")
	if err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}

	// setup config file (uses .env at root level)
	cfg := config.LoadServerConfig()
	p := pool.New(cfg.QueueSize, cfg.ConcurrentWorkers, cfg.MaxRetries)

	p.Start()

	// Initialize
	srv := server.New(p, cfg.Port, cfg.DefaultMaxTokens)
	srv.Setup()

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
