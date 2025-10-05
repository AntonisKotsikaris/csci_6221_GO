package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ChatRequest struct {
	Message string `json:"message"`
}

type LlamaRequest struct {
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LlamaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
}

func main() {
	http.HandleFunc("/chat", handleChat)

	fmt.Println("Go server running on http://localhost:9000")
	fmt.Println("Forwarding to llama.cpp at http://localhost:8080")
	err := http.ListenAndServe(":9000", nil)
	if err != nil {
		return
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq ChatRequest
	err := json.NewDecoder(r.Body).Decode(&chatReq)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received message: %s\n", chatReq.Message)

	llamaReq := LlamaRequest{
		Messages: []Message{
			{Role: "user", Content: chatReq.Message},
		},
		MaxTokens: 100,
	}

	jsonData, err := json.Marshal(llamaReq)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(
		"http://localhost:8080/v1/chat/completions",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		http.Error(w, "Failed to contact llama server", http.StatusInternalServerError)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response", http.StatusInternalServerError)
		return
	}

	var llamaResp LlamaResponse
	err = json.Unmarshal(body, &llamaResp)
	if err != nil {
		http.Error(w, "Failed to parse response", http.StatusInternalServerError)
		return
	}

	reply := ""
	if len(llamaResp.Choices) > 0 {
		reply = llamaResp.Choices[0].Message.Content
	}

	fmt.Printf("Reply: %s\n", reply)

	chatResp := ChatResponse{Reply: reply}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(chatResp)
}
