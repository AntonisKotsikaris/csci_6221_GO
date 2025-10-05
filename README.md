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

### Go Packages
Some Go packages we'll use to help complete the project faster (don't re-invent the wheel!).:
1. Gorgonia ([link](https://gorgonia.org/)). 
   This tool will help us do Deep Learning in Go (differentiation, ReLU, etc.)
2. GoNum ([link](https://www.gonum.org/)) Like NumPy but for Go :) linear algebra etc.
3. Some CLI package to interact more easily with GoLlama?
4. Etc.
## Team Responsibilities

### Team Member 1: Model Architecture - Kirtan
- Model structure definitions
- Weight loading implementation (turning huggingface into proper file types for Go)

### Team Member 2: Inference Engine - Dan
- Forward pass logic
- Optimization (Go Concurrency etc.)

### Team Member 3: Math & Data Structures - Ritvik
- Custom data structure wrappers around Gorgonia
- Utility math functions (tensor, ReLU, Softmax, etc.)
- Performance optimization

### Team Member 4: Text Processing - Tony
- Tokenization
- Named Entity Recognition (NER)
- Text preprocessing and postprocessing

### Team Member 5: Integration & DevOps - Hetel
- CLI (implement Ollama or custom)
- Testing (gofmt, go vet, golangci-lint etc.)
- Documentation and deployment