package main

import (
	"gollama/internal/config"
	"gollama/internal/pool"
	"gollama/internal/server"
	"log"
)

func main() {
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
