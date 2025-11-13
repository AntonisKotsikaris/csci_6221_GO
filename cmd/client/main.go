package main

import (
	"flag"
	"fmt"
	"gollama/internal/client"
	"log"
	"net/http"
)

func main() {
	port := flag.Int("port", 9001, "Port number for the client to run on")
	llamaPort := flag.Int("llama-port", 8080, "Port number for the llama.cpp instance")
	flag.Parse()

	//initialize and setup the client
	c := client.New(*port)
	c.Setup(*llamaPort)

	// Auto-connect to server
	go func() {
		log.Println("Attempting auto-connect to GoLlama server...")

		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/connect", *port))
		if err != nil {
			log.Printf("Auto-connect failed: %v", err)
		} else {
			resp.Body.Close()
			log.Println("Auto-connect to server successful!")
		}
	}()

	//run
	if err := c.Start(); err != nil {
		log.Fatalf("Client error: %v", err)
	}
}
