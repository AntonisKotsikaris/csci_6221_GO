package main

import (
	"gollama/internal/client"
	"log"
)

func main() {

	//initialize and setup the client
	c := client.New(9001)
	c.Setup()

	//run
	if err := c.Start(); err != nil {
		log.Fatalf("Client error: %v", err)
	}
}
