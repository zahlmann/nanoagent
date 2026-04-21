package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
)

type BashInput struct {
	Args []string `json:"args"`
}

func bashTool() responses.ToolUnionParam {
	return responses.ToolUnionParam{
		OfFunction: &responses.FunctionToolParam{
			Name:        "bash",
			Description: openai.String("Run bash command. Always include the executable first."),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"args": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
				},
				"required":             []string{"args"},
				"additionalProperties": false,
			},
		},
	}
}

func main() {
	ctx := context.Background()
	client := openai.NewClient()
	finalTurn := false
	tools := []responses.ToolUnionParam{bashTool()}

	response, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_4,
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String("Find the list of files in this directory and of the parent directory."),
		},
		Tools: tools,
	})
	if err != nil {
		log.Fatalf("openai request failed: %v", err)
	}

	for i := 0; !finalTurn; i++ {
		// Check for function calls in the response output
		finalTurn = true
		var toolOutputs []responses.ResponseInputItemUnionParam

		for _, item := range response.Output {
			if item.Type != "function_call" {
				continue
			}

			finalTurn = false
			toolCall := item.AsFunctionCall()
			fmt.Printf("tool arguments raw: %s\n", toolCall.Arguments)

			if toolCall.Name != "bash" {
				log.Fatalf("unexpected tool call: %s", toolCall.Name)
			}

			var input BashInput
			if err := json.Unmarshal([]byte(toolCall.Arguments), &input); err != nil {
				log.Fatalf("failed to parse tool arguments: %v", err)
			}
			if len(input.Args) == 0 {
				log.Fatal("bash command failed: no args provided")
			}

			bashOut, err := exec.Command(input.Args[0], input.Args[1:]...).CombinedOutput()
			if err != nil {
				log.Fatalf("bash command failed: %s", err)
			}
			bashOutString := string(bashOut)
			fmt.Printf("Output from bash command:\n%s\n", bashOutString)

			toolOutputs = append(toolOutputs, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					CallID: toolCall.CallID,
					Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
						OfString: openai.String(bashOutString),
					},
				},
			})
		}

		if finalTurn {
			continue
		}

		response, err = client.Responses.New(ctx, responses.ResponseNewParams{
			Model:              openai.ChatModelGPT5_4,
			PreviousResponseID: openai.String(response.ID),
			Input: responses.ResponseNewParamsInputUnion{
				OfInputItemList: toolOutputs,
			},
			Tools: tools,
		})
		if err != nil {
			log.Fatalf("openai request failed: %v", err)
		}
	}
	fmt.Printf("********* Final response **********\n%s\n***********************************\n", response.OutputText())
}
