# GoLlama
This repository contains the project developed in Go for the course Advanced Software Paradigms (CSCI 6221).

## Quick Start

### Prerequisites

- Go 1.25 installed
- Git
- (Mac) Xcode Command Line Tools
- (Mac) Homebrew
- (Windows) Visual Studio with C++ support or MinGW

### Setup Instructions

#### 1. Install Build Tools

**macOS:**
```bash
# Install Xcode Command Line Tools
xcode-select --install

# Install cmake
brew install cmake
```

**Windows:**
```bash
# Using Chocolatey
choco install cmake git

# Or download CMake from: https://cmake.org/download/
# And Git from: https://git-scm.com/downloads/win
```

#### 2. Clone and Build llama.cpp

```bash
# Clone llama.cpp
git clone https://github.com/ggml-org/llama.cpp
cd llama.cpp

# Build it
cmake -B build
cmake --build build --config Release
```

**Note for Mac:** Metal GPU acceleration is enabled by default  
**Note for Windows:** This builds CPU version. For GPU, see [llama.cpp docs](https://github.com/ggml-org/llama.cpp/blob/master/docs/build.md)

#### 3. Download a Model

Create a models directory and download a small model for testing:

```bash
# Create models directory
mkdir -p models
cd models

# Download Qwen 0.5B (small, ~300MB, good for testing)
curl -L -o qwen-0.5b.Q4_K_M.gguf https://huggingface.co/Qwen/Qwen2-0.5B-Instruct-GGUF/resolve/main/qwen2-0_5b-instruct-q4_k_m.gguf

# OR download TinyLlama 1.1B (~900MB)
curl -L -o tinyllama-1.1b.Q4_K_M.gguf https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf

# Return to llama.cpp directory
cd ..
```

#### 4. Start llama.cpp Server

```bash
# From llama.cpp directory
./build/bin/llama-server -m ./models/qwen-0.5b.Q4_K_M.gguf --host 0.0.0.0 --port 8080
```

You should see:
```
llama server listening at http://0.0.0.0:8080
```

#### 5. Test llama.cpp is Working

Open a new terminal and test:

```bash
# Check health
curl http://localhost:8080/health

# Test completion
curl http://localhost:8080/v1/completions \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is 2+2?",
    "max_tokens": 50
  }'
```

Or open your browser to: `http://localhost:8080`

#### 6. Clone and Run Gollama

```bash
# Clone this repo
git clone https://github.com/AntonisKotsikaris/csci_6221_GO.git
cd csci_6221_GO

# Run the server
go run main.go
```

The Gollama server should start on `http://localhost:9000`

#### 7. Test the Gollama API

Using curl:
```bash
curl -X POST http://localhost:9000/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Tell me a joke"
  }'
```

Or use Insomnia/Postman:
- **URL:** `POST http://localhost:9000/chat`
- **Headers:** `Content-Type: application/json`
- **Body:**
  ```json
  {
    "message": "Hello!"
  }
  ```