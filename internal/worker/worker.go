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
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/health", clientPort))
	if err != nil || resp.StatusCode != http.StatusOK {
		http.Error(writer, "Cannot register: worker unhealthy", http.StatusServiceUnavailable)
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
		"http://localhost:9000/connectWorker",
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
}

func handleExecute(writer http.ResponseWriter, request *http.Request) {
	//basically just pass the request from the server into llama.cpp. So we don't handle endpoint names etc.
	//That's all done in the server. This should just pass the request from the server into llama.cpp using
	//this worker as the middle-man.
	//
	//Applied the command OODP pattern here: execute handles ALL
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
