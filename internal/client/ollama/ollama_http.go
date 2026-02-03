package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaHTTPClient interface {
	GetBaseURL() string
	ChatCompletion(ctx context.Context, request *ChatRequest) (*ChatResponse, error)
	ChatCompletionStream(ctx context.Context, request *ChatRequest, callback func(*ChatResponse)) error
}

type ollamaHTTPClient struct {
	baseURL string
	client  *http.Client
}

func NewOllamaHTTPClient(baseURL string) *ollamaHTTPClient {
	return &ollamaHTTPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (c *ollamaHTTPClient) GetBaseURL() string {
	return c.baseURL
}

func (c *ollamaHTTPClient) ChatCompletion(ctx context.Context, request *ChatRequest) (*ChatResponse, error) {
	request.Stream = false

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// fmt.Printf("Body: %s", string(jsonData))

	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("ERROR: failed to request chat completion: %s\n", string(body))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (c *ollamaHTTPClient) ChatCompletionStream(ctx context.Context, request *ChatRequest, callback func(*ChatResponse)) error {
	request.Stream = true

	jsonData, err := json.Marshal(request)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/api/chat", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("ERROR: failed to request chat completion: %s\n", string(body))
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var response ChatResponse
		if err := decoder.Decode(&response); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		callback(&response)

		if response.Message.Done {
			break
		}
	}

	return nil
}
