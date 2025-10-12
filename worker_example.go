package main

import (
	"fmt"
	"time"
)

type Request struct {
	ID      string
	Message string
	ReplyCh chan string
}

type WorkerPool struct {
	jobs       chan Request
	numWorkers int
}

func NewWorkerPool(numWorkers int, queueSize int) *WorkerPool {
	return &WorkerPool{
		jobs:       make(chan Request, queueSize),
		numWorkers: numWorkers,
	}
}

func (wp *WorkerPool) Start() {
	for i := 1; i <= wp.numWorkers; i++ {
		go wp.worker(i)
	}
}

func (wp *WorkerPool) worker(id int) {
	for req := range wp.jobs {
		fmt.Printf("[%s] Worker %d processing: %s\n", time.Now().Format("15:04:05"), id, req.ID)
		time.Sleep(2 * time.Second)
		result := fmt.Sprintf("Response to: %s", req.Message)
		req.ReplyCh <- result
	}
}

func (wp *WorkerPool) Submit(req Request) {
	wp.jobs <- req
}

func main() {
	fmt.Printf("Start time: [%s]\n", time.Now().Format("15:04:05"))

	pool := NewWorkerPool(3, 10)
	pool.Start()

	replyChannels := make([]chan string, 3)

	for i := 0; i < 3; i++ {
		replyChannels[i] = make(chan string)

		req := Request{
			ID:      fmt.Sprintf("req-%d", i+1),
			Message: fmt.Sprintf("Question %d", i+1),
			ReplyCh: replyChannels[i],
		}

		pool.Submit(req)
	}

	for i := 0; i < 3; i++ {
		response := <-replyChannels[i]
		fmt.Printf("Got response: %s\n", response)
	}
	fmt.Printf("End time: [%s]\n", time.Now().Format("15:04:05"))
}
