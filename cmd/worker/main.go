package main

import (
	"flag"
	"fmt"
	"gollama/internal/worker"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 9001, "Port number for the worker to run on")
	llamaPort := flag.Int("llama-port", 8080, "Port number for the llama.cpp instance")
	flag.Parse()

	//initialize and setup the worker
	c := worker.New(*port)
	c.Setup(*llamaPort)

	// Start worker server first (non-blocking)
	go func() {
		if err := c.Start(); err != nil {
			log.Fatalf("Worker server error: %v", err)
		}
	}()
	// autoConnect(port)
	select {}
}

func autoConnect(port *int) {
	log.Println("Attempting auto-connect to GoLlama server...")
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/connect", *port))
	if err != nil {
		log.Fatalf("Auto-connect failed: %v - shutting down", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Auto-connect failed: worker returned status %d - shutting down", resp.StatusCode)
	}

	log.Println("Auto-connect to server successful!")
}
