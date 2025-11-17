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
	pool             *pool.Pool
	port             int
	defaultMaxTokens int
}

/*
New creates a new server instance
*/
func New(p *pool.Pool, port int, defaultMaxTokens int) *Server {
	return &Server{
		pool:             p,
		port:             port,
		defaultMaxTokens: defaultMaxTokens,
	}
}

/*
Setup configures all routes and starts the server
TODO: Add other endpoints: NER, summarize, etc. endpoints
*/
func (s *Server) Setup() {
	// Register authentication endpoints
	http.HandleFunc("/auth/token", handler.HandleGetToken())

	// Register handlers
	http.HandleFunc("/connectWorker", handler.HandleConnectWorker(s.pool))
	http.HandleFunc("/chat", handler.HandleChat(s.pool, s.defaultMaxTokens))

	// Register public handlers
	http.HandleFunc("/health", handler.HandleHealth(s.pool))
	http.HandleFunc("/stats", handler.HandleStats(s.pool))

	log.Printf("GoLlama server running on http://localhost:%d", s.port)
	log.Println("Forwarding to llama.cpp workers")
	log.Printf("  POST /auth/token - Get JWT token for worker")
	log.Printf("  POST /connectWorker - Register a new worker (requires JWT)")
	log.Printf("  POST /chat - Submit a chat message (requires JWT)")
	log.Printf("  GET  /health - Check server health")
	log.Printf("  GET  /stats - View worker statistics")
}

// Start begins listening for requests
func (s *Server) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}
