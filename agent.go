package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const endpoint string = "https://api.openai.com/v1/chat/completions"

var oaiKey = os.Getenv("OPENAI_API_KEY")

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type tool struct {
	Type     string `json:"type"`
	Function struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Parameters  struct {
			Type       string         `json:"type"`
			Properties map[string]any `json:"properties"`
			Required   []string       `json:"required"`
		} `json:"parameters"`
	} `json:"function"`
}

type apiInput struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []tool    `json:"tools"`
}

func main() {
	input := apiInput{
		Model: "gpt-5.4",
		Messages: []message{
			{
				Role:    "developer",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello!",
			},
		},
	}
	type chatResponse struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	var inputBuf bytes.Buffer
	err := json.NewEncoder(&inputBuf).Encode(input)
	if err != nil {
		log.Fatalf("encoding of api input failed: %v", err)
	}
	req, err := http.NewRequest("POST", endpoint, &inputBuf)
	if err != nil {
		log.Fatalf("creation of request to openai failed: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+oaiKey)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("request to openai failed: %v", err)
	}
	defer resp.Body.Close()
	var out chatResponse
	json.NewDecoder(resp.Body).Decode(&out)
	fmt.Printf("model response: %v", out)
}
