package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const oaiEndpoint string = "https://api.openai.com/v1/chat/completions"
const dsEndpoint string = "https://api.deepseek.com/chat/completions"
const oaiModel string = "gpt-5.5"
const dsModel string = "deepseek-v4-pro"

var oaiKey = os.Getenv("OPENAI_API_KEY")
var dsKey = os.Getenv("DEEPSEEK_API_KEY")

type message struct {
	Role             string     `json:"role"`
	Content          *string    `json:"content,omitempty"`
	ToolCallID       string     `json:"tool_call_id,omitempty"`
	ToolCalls        []toolCall `json:"tool_calls,omitempty"`
	ReasoningContent *string    `json:"reasoning_content,omitempty"`
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

type modelFunctionArgs struct {
	Args []string `json:"args"`
}

type toolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function functionCall `json:"function"`
}

type chatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Choices []struct {
		Message      message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: go run agent.go [openai, deepseek] 'prompt'")
		return
	}
	var model string
	var endpoint string
	var apiKey string
	switch providerArg := os.Args[1]; providerArg {
	case "openai":
		model = oaiModel
		endpoint = oaiEndpoint
		apiKey = oaiKey
	case "deepseek":
		model = dsModel
		endpoint = dsEndpoint
		apiKey = dsKey
	default:
		fmt.Println("usage: go run agent.go [openai, deepseek] 'prompt'")
		return
	}
	apiInputPtr := constructApiInput("You are a helpful assistant.", model)
	prompt := os.Args[2]
	addUserMessage(apiInputPtr, prompt)
	outMessage, finishReason, err := modelCall(apiInputPtr, endpoint, apiKey)
	if err != nil {
		log.Fatalf("failure: %v", err)
	}
	fmt.Printf("model response: %+v\n", outMessage)
	for finishReason != "stop" {
		addAssistantMessage(apiInputPtr, outMessage)
		toolMessages := executeBash(outMessage.ToolCalls)
		addToolResults(apiInputPtr, toolMessages)
		outMessage, finishReason, err = modelCall(apiInputPtr, endpoint, apiKey)
		if err != nil {
			log.Fatalf("failure: %v", err)
		}
		fmt.Printf("model response: %+v\n", outMessage)
	}
	fmt.Printf("final response:\n%+v\n", *outMessage.Content)
}

func constructApiInput(developerMessage string, model string) *apiInput {
	return &apiInput{
		Model: model,
		Messages: []message{
			{
				Role:    "system",
				Content: &developerMessage,
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
}

func addUserMessage(apiInput *apiInput, userMessage string) {
	apiInput.Messages = append(
		apiInput.Messages,
		message{
			Role:    "user",
			Content: &userMessage,
		},
	)
}

func addAssistantMessage(apiInput *apiInput, assistantMessage message) {
	apiInput.Messages = append(apiInput.Messages, assistantMessage)
}

func addToolResults(apiInput *apiInput, toolMessages []message) {
	apiInput.Messages = append(apiInput.Messages, toolMessages...)
}

func executeBash(toolCalls []toolCall) []message {
	var toolMessages []message
	for _, call := range toolCalls {
		var args modelFunctionArgs
		json.Unmarshal([]byte(call.Function.Arguments), &args)
		cmd, _ := exec.Command(string(args.Args[0]), args.Args[1:]...).CombinedOutput()
		result := string(cmd)
		toolMessages = append(
			toolMessages,
			message{
				Role:       "tool",
				ToolCallID: call.ID,
				Content:    &result,
			},
		)
	}
	return toolMessages
}

func modelCall(apiInput *apiInput, endpoint string, apiKey string) (message, string, error) {
	var inputBuf bytes.Buffer
	json.NewEncoder(&inputBuf).Encode(apiInput)
	req, err := http.NewRequest("POST", endpoint, &inputBuf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return message{}, "", err
	}
	defer resp.Body.Close()
	var out chatResponse
	json.NewDecoder(resp.Body).Decode(&out)
	return out.Choices[0].Message, out.Choices[0].FinishReason, nil
}
