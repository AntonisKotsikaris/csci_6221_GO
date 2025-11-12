package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	auth "gollama/internal/auth"
)

var clientPort int

/*
Client manages the HTTP client that connects to llama.cpp
*/
type Client struct {
	port int
}

// New initializes the client object
func New(port int) *Client {
	clientPort = port // Store in package variable
	return &Client{
		port: port,
	}
}

/*
Start to run the client.
*/
func (c *Client) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", c.port), nil)
}

func (c *Client) Setup() {
	// Register auth handlers
	auth.Setup()

	// Register protected handlers with auth middleware
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/connect", handleConnectToServer)
	http.HandleFunc("/execute", handleExecute)

	log.Printf("GoLlama client running on http://localhost:%d", c.port)
	log.Printf("  GET /login - Login page")
	log.Printf("  GET /signup - Signup page")
	log.Printf("  GET /health - Check client health")
	log.Printf("  GET /connect - Connect to server (protected)")
	log.Printf("  POST /execute - Execute commands (protected)")
}

func handleHealth(writer http.ResponseWriter, request *http.Request) {
	// Ping llama.cpp instance (assuming it's on localhost:8080)
	resp, err := http.Get("http://localhost:8080/health")
	if err != nil {
		http.Error(writer, "llama.cpp unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(writer, "Llama.cpp unhealthy", http.StatusServiceUnavailable)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	log.Printf("  GET /health - OK")
	fmt.Fprintf(writer, `{"status":"ok"}`)
}

func handleConnectToServer(writer http.ResponseWriter, request *http.Request) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", clientPort))
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(writer, "Cannot register: client unhealthy", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Prepare registration payload
	clientURL := fmt.Sprintf("http://localhost:%d", clientPort)
	workerInfo := map[string]string{
		"url":   clientURL,
		"model": "test-model", // TODO: Make this based on the llama.cpp actual model
	}

	// prepare payload
	payload, err := json.Marshal(workerInfo)
	if err != nil {
		http.Error(writer, "Registration failed", http.StatusInternalServerError)
		return
	}

	// post to server
	serverResp, err := http.Post(
		"https://jaquelyn-cuneal-undepressively.ngrok-free.dev/connectWorker",
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		http.Error(writer, "Server unreachable", http.StatusBadGateway)
		return
	}
	defer serverResp.Body.Close()

	if serverResp.StatusCode != http.StatusOK {
		http.Error(writer, "Server rejected registration", http.StatusBadGateway)
		return
	}

	// Echo server's response back to caller
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, serverResp.Body)

	log.Printf("Successfully connected to server")
}

func handleExecute(writer http.ResponseWriter, request *http.Request) {
	//basically just pass the request from the server into llama.cpp. So we don't handle endpoint names etc.
	//That's all done in the server. This should just pass the request from the server into llama.cpp using
	//this client as the middle-man.
	var executeReq struct {
		Endpoint string          `json:"endpoint"`
		Body     json.RawMessage `json:"body"`
	}
	err := json.NewDecoder(request.Body).Decode(&executeReq)
	if err != nil {
		http.Error(writer, "Invalid request", http.StatusBadRequest)
		return
	}
	defer request.Body.Close()

	if executeReq.Endpoint == "" {
		http.Error(writer, "Missing endpoint", http.StatusBadRequest)
		return
	}

	//dynamically create the endpoint based on the request data from Gollama server
	resp, err := http.Post(
		fmt.Sprintf("http://localhost:8080%s", executeReq.Endpoint),
		"application/json",
		bytes.NewReader(executeReq.Body),
	)
	if err != nil {
		http.Error(writer, "llama.cpp unavailable", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(writer, "Error reading llama.cpp response", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(resp.StatusCode)
	writer.Write(body)
}
