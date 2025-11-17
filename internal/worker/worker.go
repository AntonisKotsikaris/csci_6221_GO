package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

var clientPort int
var llamaPort int
var busyFlag bool
var cachedToken string // Store the JWT token for reuse

/*
Client manages the HTTP worker that connects to llama.cpp
*/
type Client struct {
	port int
}

// New initializes the worker object
func New(port int) *Client {
	clientPort = port // Store in package variable
	return &Client{
		port: port,
	}
}

/*
Start to run the worker.
*/
func (c *Client) Start() error {
	return http.ListenAndServe(fmt.Sprintf(":%d", c.port), nil)
}

func (c *Client) Setup(llamaPortArg int) {
	llamaPort = llamaPortArg // Store in package variable

	// Register handlers
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/connect", handleConnectToServer)
	http.HandleFunc("/execute", handleExecute)

	log.Printf("GoLlama worker running on http://localhost:%d", c.port)
	log.Printf("Connecting to llama.cpp on port %d", llamaPort)
	log.Printf("  GET /health - Check worker health")
	log.Printf("  GET /connect - Connect to server")
}

func handleHealth(writer http.ResponseWriter, request *http.Request) {
	// Ping llama.cpp instance
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", llamaPort))
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
	fmt.Fprintf(writer, `{"busy":"%v"}`, busyFlag)
}

func handleConnectToServer(writer http.ResponseWriter, request *http.Request) {
	// Parse request body for credentials
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	err := json.NewDecoder(request.Body).Decode(&credentials)
	if err != nil {
		http.Error(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate credentials are provided
	if credentials.Username == "" || credentials.Password == "" {
		http.Error(writer, "Username and password are required", http.StatusBadRequest)
		return
	}

	// Check worker health
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", clientPort))
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(writer, "Cannot register: worker unhealthy", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Step 1: Get JWT token from server
	workerID := fmt.Sprintf("worker-%d", clientPort)
	clientURL := fmt.Sprintf("http://localhost:%d", clientPort)

	tokenReq := map[string]string{
		"worker_id": workerID,
		"url":       clientURL,
		"username":  credentials.Username, // Use from request body
		"password":  credentials.Password, // Use from request body
	}

	tokenPayload, err := json.Marshal(tokenReq)
	if err != nil {
		http.Error(writer, "Token request preparation failed", http.StatusInternalServerError)
		return
	}

	tokenResp, err := http.Post(
		"http://localhost:9000/auth/token",
		"application/json",
		bytes.NewReader(tokenPayload),
	)
	if err != nil {
		http.Error(writer, "Cannot get token from server", http.StatusBadGateway)
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		log.Printf("Token request failed: %d - %s", tokenResp.StatusCode, string(body))
		http.Error(writer, "Server rejected token request", http.StatusBadGateway)
		return
	}

	var tokenData struct {
		Token string `json:"token"`
	}
	err = json.NewDecoder(tokenResp.Body).Decode(&tokenData)
	if err != nil {
		http.Error(writer, "Invalid token response", http.StatusInternalServerError)
		return
	}

	// Cache the token for reuse in chat requests
	cachedToken = tokenData.Token

	// Step 2: Register with server using JWT token
	workerInfo := map[string]string{
		"url":   clientURL,
		"model": "test-model", // TODO: Make this based on the llama.cpp actual model
	}

	payload, err := json.Marshal(workerInfo)
	if err != nil {
		http.Error(writer, "Registration failed", http.StatusInternalServerError)
		return
	}

	// Create request with authorization header
	req, err := http.NewRequest("POST", "http://localhost:9000/connectWorker", bytes.NewReader(payload))
	if err != nil {
		http.Error(writer, "Request creation failed", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokenData.Token))

	client := &http.Client{}
	serverResp, err := client.Do(req)
	if err != nil {
		http.Error(writer, "Server unreachable", http.StatusBadGateway)
		return
	}
	defer serverResp.Body.Close()

	if serverResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(serverResp.Body)
		log.Printf("Registration rejected: %d - %s", serverResp.StatusCode, string(body))
		http.Error(writer, "Server rejected registration", http.StatusBadGateway)
		return
	}

	// Echo server's response back to caller
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	io.Copy(writer, serverResp.Body)
	log.Printf("Worker %s registered successfully with token", workerID)
}

func handleExecute(writer http.ResponseWriter, request *http.Request) {
	//basically just pass the request from the server into llama.cpp. So we don't handle endpoint names etc.
	//That's all done in the server. This should just pass the request from the server into llama.cpp using
	//this worker as the middle-man.
	var executeReq struct {
		Endpoint string          `json:"endpoint"`
		Body     json.RawMessage `json:"body"`
	}
	busyFlag = true

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
		fmt.Sprintf("http://localhost:%d%s", llamaPort, executeReq.Endpoint),
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

	log.Printf("Executed task at endpoint: %s", executeReq.Endpoint)
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(resp.StatusCode)
	writer.Write(body)
	busyFlag = false
}
