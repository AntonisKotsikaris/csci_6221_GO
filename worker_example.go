package main

import (
	"fmt"
	"time"
)

type Request struct {
	ID      string
	Message string
	ReplyCh chan string // Channel to send response back
}

// 2. Worker processes requests from a queue
func worker(id int, jobs <-chan Request) {
	for req := range jobs {
		fmt.Printf("[%s] Worker %d processing: %s\n", time.Now().Format("15:04:05"), id, req.ID)
		time.Sleep(2 * time.Second)
		result := fmt.Sprintf("Response to: %s", req.Message)
		req.ReplyCh <- result
	}
}

func main() {
	fmt.Printf("Start time: [%s]\n", time.Now().Format("15:04:05"))
	jobs := make(chan Request, 10)

	// we need to convert this into a POOL of workers - instead of being a local function, it's a pool of
	//llama.cpp backend instances
	go worker(1, jobs)
	go worker(2, jobs)
	go worker(3, jobs)
	go worker(4, jobs)
	go worker(5, jobs)

	replyChannels := make([]chan string, 3)

	for i := 0; i < 3; i++ {
		replyChannels[i] = make(chan string)

		req := Request{
			ID:      fmt.Sprintf("req-%d", i+1),
			Message: fmt.Sprintf("Question %d", i+1),
			ReplyCh: replyChannels[i],
		}

		jobs <- req
	}

	for i := 0; i < 3; i++ {
		response := <-replyChannels[i]
		fmt.Printf("Got response: %s\n", response)
	}
	fmt.Printf("End time: [%s]\n", time.Now().Format("15:04:05"))
}
