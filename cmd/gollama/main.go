package main

import (
	"gollama/internal/pool"
	"gollama/internal/server"
	"log"
)

func main() {
	//initialize and start the worker pool
	p := pool.New(5000)
	p.Start()

	//initialize and setup the server with the created worker pool
	srv := server.New(p, 9000)
	srv.Setup()

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
