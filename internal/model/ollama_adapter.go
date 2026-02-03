package model

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"net/http"

	"github.com/lopesgabriel/adk-go/internal/client/ollama"
	modelx "google.golang.org/adk/model"
	"google.golang.org/adk/tool"
	"google.golang.org/genai"
)

type FunctionTool interface {
	tool.Tool
	Declaration() *genai.FunctionDeclaration
	Description() string
}

type ollamaModelAdapter struct {
	ModelName string
	client    ollama.OllamaHTTPClient
}

func NewOllamaModelAdapter(modelName string, client ollama.OllamaHTTPClient) *ollamaModelAdapter {
	return &ollamaModelAdapter{
		ModelName: modelName,
		client:    client,
	}
}

func (m *ollamaModelAdapter) Name() string {
	return m.ModelName
}

func (m *ollamaModelAdapter) GenerateContent(ctx context.Context, req *modelx.LLMRequest, stream bool) iter.Seq2[*modelx.LLMResponse, error] {
	m.maybeAppendUserContent(req)
	if req.Config == nil {
		req.Config = &genai.GenerateContentConfig{}
	}
	if req.Config.HTTPOptions == nil {
		req.Config.HTTPOptions = &genai.HTTPOptions{}
	}
	if req.Config.HTTPOptions.Headers == nil {
		req.Config.HTTPOptions.Headers = make(http.Header)
	}

	if stream {
		return m.generateStream(ctx, req)
	}

	return func(yield func(*modelx.LLMResponse, error) bool) {
		resp, err := m.generate(ctx, req)
		yield(resp, err)
	}
}

func (m *ollamaModelAdapter) generate(ctx context.Context, req *modelx.LLMRequest) (*modelx.LLMResponse, error) {
	chatResp, err := m.client.ChatCompletion(ctx, m.adkToOllama(req))
	if err != nil {
		return nil, err
	}

	resp := m.ollamaToADK(chatResp)
	return resp, nil
}

func (m *ollamaModelAdapter) generateStream(ctx context.Context, req *modelx.LLMRequest) iter.Seq2[*modelx.LLMResponse, error] {
	return func(yield func(*modelx.LLMResponse, error) bool) {
		err := m.client.ChatCompletionStream(ctx, m.adkToOllama(req), func(chatResp *ollama.ChatResponse) {
			resp := m.ollamaToADK(chatResp)
			yield(resp, nil)
		})
		if err != nil {
			yield(nil, err)
			return
		}
	}
}

func (m *ollamaModelAdapter) adkToOllama(req *modelx.LLMRequest) *ollama.ChatRequest {
	ollamaReq := &ollama.ChatRequest{
		Model:    m.ModelName,
		Messages: []ollama.RequestMessage{},
		Tools:    []ollama.AvailableTool{},
		Think:    true,
	}

	if req.Config != nil && req.Config.SystemInstruction != nil {
		for _, part := range req.Config.SystemInstruction.Parts {
			ollamaReq.Messages = append(ollamaReq.Messages, ollama.RequestMessage{
				Role:    ollama.RoleSystem,
				Content: part.Text,
			})
		}
	}

	for name, tool := range req.Tools {
		declarations := tool.(FunctionTool).Declaration()
		b, _ := json.Marshal(declarations.ParametersJsonSchema)
		var paramsMap map[string]any
		if err := json.Unmarshal(b, &paramsMap); err != nil {
			fmt.Printf("failed to unmarshal params: %v", err)
			continue
		}

		ollamaReq.Tools = append(ollamaReq.Tools, ollama.AvailableTool{
			Type: ollama.ToolTypeFunction,
			Function: ollama.FunctionTool{
				Name:        name,
				Description: tool.(FunctionTool).Description(),
				Parameters:  paramsMap,
			},
		})
	}

	for _, content := range req.Contents {
		for _, part := range content.Parts {
			if part.Text != "" {
				ollamaReq.Messages = append(ollamaReq.Messages, ollama.RequestMessage{
					Role:    ollama.Role(content.Role),
					Content: part.Text,
				})
			}

			if part.FunctionResponse != nil {
				functionResponseContent, err := json.Marshal(part.FunctionResponse.Response)
				if err != nil {
					break
				}

				ollamaReq.Messages = append(ollamaReq.Messages, ollama.RequestMessage{
					Role:    ollama.RoleTool,
					Content: string(functionResponseContent),
				})
			}

			if part.FunctionCall != nil {
				argumentsContent, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					break
				}

				ollamaReq.Messages = append(ollamaReq.Messages, ollama.RequestMessage{
					Role: ollama.RoleUser,
					ToolCalls: []ollama.ToolCall{
						{
							Function: ollama.FunctionTool{
								Name: part.FunctionCall.Name,
							},
						},
					},
					Content: string(argumentsContent),
				})
			}
		}
	}

	return ollamaReq
}

func (m *ollamaModelAdapter) ollamaToADK(resp *ollama.ChatResponse) *modelx.LLMResponse {
	respContent := genai.NewContentFromText(resp.Message.Content, genai.RoleModel)

	if resp.Message.Thinking != "" {
		respContent.Parts = append(respContent.Parts, &genai.Part{
			Thought: true,
			Text:    resp.Message.Thinking,
		})
	}

	for _, toolCall := range resp.Message.ToolCalls {
		respContent.Parts = append(respContent.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				Name: toolCall.Function.Name,
				Args: toolCall.Function.Arguments,
			},
		})
		resp.Message.Done = true
	}

	return &modelx.LLMResponse{
		Content:      respContent,
		TurnComplete: resp.Message.Done,
	}
}

// maybeAppendUserContent appends a user content, so that model can continue to output.
func (m *ollamaModelAdapter) maybeAppendUserContent(req *modelx.LLMRequest) {
	if len(req.Contents) == 0 {
		req.Contents = append(req.Contents, genai.NewContentFromText("Handle the requests as specified in the System Instruction.", "user"))
	}

	if last := req.Contents[len(req.Contents)-1]; last != nil && last.Role != "user" {
		req.Contents = append(req.Contents, genai.NewContentFromText("Continue processing previous requests as instructed. Exit or provide a summary if no more outputs are needed.", "user"))
	}
}
