package internal

import "time"

/*
Message represents a single message in the conversation
*/
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

/*
ChatRequest is what users send to GoLlama which is then passed to GoLlama spokes or workers
*/
type ChatRequest struct {
	Message string `json:"message"`
}

/*
ChatResponse is what GoLlama returns to clients.
*/
type ChatResponse struct {
	Reply string `json:"reply"`
}

/*
LlamaRequest is what we send to llama.cpp
TODO: We're not handling chat history yet. But the string of messages is the basic idea I think.
*/
type LlamaRequest struct {
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
}

/*
LlamaResponse is what llama.cpp returns
TODO: Right now it's just a stub of the response. Here is a full llama.cpp response example:

	{
		"choices": [
			{
				"finish_reason": "stop",
				"index": 0,
				"message": {
					"role": "assistant",
					"content": "Sure, here's a joke for you:\n\nWhy was the math book sad?\n\nBecause it had too many problems."
				}
			}
		],
		"created": 1762115301,
		"model": "gpt-3.5-turbo",
		"system_fingerprint": "b6691-35266573",
		"object": "chat.completion",
		"usage": {
			"completion_tokens": 24,
			"prompt_tokens": 15,
			"total_tokens": 39
		},
		"id": "chatcmpl-hUaZPeUpyUcyxprCM5rD67vGARVVyoNt",
		"timings": {
			"cache_n": 0,
			"prompt_n": 15,
			"prompt_ms": 52.9,
			"prompt_per_token_ms": 3.5266666666666664,
			"prompt_per_second": 283.55387523629486,
			"predicted_n": 24,
			"predicted_ms": 337.327,
			"predicted_per_token_ms": 14.055291666666667,
			"predicted_per_second": 71.14758083402751
		}
	}
*/
type LlamaResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error string `json:"error,omitempty"`
}

/*
WorkerStats tracks performance metrics for a worker
*/
type WorkerStats struct {
	ID            string    `json:"id"`
	URL           string    `json:"url"`
	JobsCompleted int       `json:"jobs_completed"`
	JobsFailed    int       `json:"jobs_failed"`
	StartTime     time.Time `json:"start_time"`
	AvgResponseMS float64   `json:"avg_response_ms"`
	Requests      int       `json:"requests"`
	LastActive    time.Time `json:"last_active"`
	Healthy       bool      `json:"healthy"`
}

/*
WorkerJob represents a request to be processed by a worker
*/
type WorkerJob struct {
	Request    LlamaRequest
	ReplyCh    chan string
	WorkerURL  string
	RetryCount int
	MaxRetries int
}

/*
SummarizeRequest is what users send to the /summarize endpoint
*/
type SummarizeRequest struct {
	Text string `json:"text"`
}

/*
SummarizeResponse is what /summarize returns to clients
*/
type SummarizeResponse struct {
	Summary string `json:"summary"`
}

/*
TranslateRequest is what users send to the /translate endpoint
*/
type TranslateRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

/*
TranslateResponse is what /translate returns to clients
*/
type TranslateResponse struct {
	Translation string `json:"translation"`
}

/*
SentimentRequest is what users send to the /sentiment endpoint
*/
type SentimentRequest struct {
	Text string `json:"text"`
}

/*
SentimentResponse is what /sentiment returns to clients
*/
type SentimentResponse struct {
	Sentiment string `json:"sentiment"`
}
