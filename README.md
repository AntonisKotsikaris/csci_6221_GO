# GoLlama
This repository contains the project developed in Go for the course Advanced Software Paradigms (CSCI 6221).

## Project Goals
1. Build a working implementation of llama cpp but for the Golang library.
2. Benchmarking: Compare inference time of GoLlama vs. Llama cpp
3. Integration testing
4. Code review and quality metrics

## High-level Architecture & Structure

Input Text → Tokenizer → Model → Inference → Output Text

The directory is structured as follows:
```
gollama/
├── cmd/gollama/main.go          # CLI application
├── pkg/
│   ├── model/                   # Model loading & architecture - team member 1
│   ├── inference/               # Forward pass & text generation (using math package) - team member 2
│   ├── math/                    # Linear algebra wrappers - team member 3
│   ├── tokenizer/               # Text processing & NER - team member 4
│   └── api/                     # CLI & Ollama integration - team member 5
└── models/                      # Model files saved from huggingface etc.
```

## Team Responsibilities

### Team Member 1: Model Architecture - ???
- Model structure definitions
- Weight loading implementation (turning huggingface into proper file types for Go)

### Team Member 2: Inference Engine - ???
- Forward pass logic
- Optimization (Go Concurrency etc.)

### Team Member 3: Math & Data Structures - ???
- Custom data structure wrappers around Gorgonia
- Utility math functions (tensor, ReLU, Softmax, etc.)
- Performance optimization

### Team Member 4: Text Processing - ???
- Tokenization
- Named Entity Recognition (NER)
- Text preprocessing and postprocessing

### Team Member 5: Integration & DevOps - ???
- CLI (implement Ollama or custom)
- Testing (gofmt, go vet, golangci-lint etc.)
- Documentation and deployment