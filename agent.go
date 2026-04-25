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

type functionParameters struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

type functionDef struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Parameters  functionParameters `json:"parameters"`
}

type tool struct {
	Type     string      `json:"type"`
	Function functionDef `json:"function"`
}

type apiInput struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Tools    []tool    `json:"tools"`
}

type functionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolCall struct {
	Id       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Role      string     `json:"role"`
			Content   string     `json:"content"`
			ToolCalls []toolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
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
				Content: "List the files in this directory and the parent directory.",
			},
		},
		Tools: []tool{
			{
				Type: "function",
				Function: functionDef{
					Name:        "bash_tool",
					Description: "Execute bash commands",
					Parameters: functionParameters{
						Type: "object",
						Properties: map[string]any{
							"args": map[string]any{
								"type": "array",
								"items": map[string]any{
									"type": "string",
								},
								"description": "Command and arguments to execute.",
							},
						},
						Required: []string{"args"},
					},
				},
			},
		},
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
