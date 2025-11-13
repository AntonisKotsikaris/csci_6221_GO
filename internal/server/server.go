package server

import (
	"fmt"
	"log"
	"net/http"

	"gollama/internal/handler"
	"gollama/internal/pool"
)

/*
Server manages the HTTP server and routes
*/
type Server struct {
	pool *pool.Pool
	port int
}

/*
New creates a new server instance
*/
func New(p *pool.Pool, port int) *Server {
	return &Server{
		pool: p,
		port: port,
	}
}

/*
Setup configures all routes and starts the server
TODO: Add other endpoints: NER, summarize, etc. endpoints
*/
func (s *Server) Setup() {
	// Register handlers
	http.HandleFunc("/connectWorker", handler.HandleConnectWorker(s.pool))
	http.HandleFunc("/chat", handler.HandleChat(s.pool))
	http.HandleFunc("/health", handler.HandleHealth(s.pool))
	http.HandleFunc("/stats", handler.HandleStats(s.pool))

	log.Printf("GoLlama server running on http://localhost:%d", s.port)
	log.Println("Forwarding to llama.cpp workers")
	log.Printf("  POST /chat - Submit a chat message")
	log.Printf("  POST /connectWorker - Register a new worker")
	log.Printf("  GET  /health - Check server health")
	log.Printf("  GET  /stats - View worker statistics")
}

// Start begins listening for requests
func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
