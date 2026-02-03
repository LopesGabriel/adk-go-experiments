package tool

import (
	"fmt"
	"io"
	"net/http"

	"github.com/google/jsonschema-go/jsonschema"
	toolx "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type WebpageTopicInput struct {
	URL string `json:"url"`
}

func NewWebpageTopicTool() (toolx.Tool, error) {
	inputSchema, _ := jsonschema.For[WebpageTopicInput](nil)

	return functiontool.New(functiontool.Config{
		Name:                        "get_webpage",
		Description:                 "This tool access a webpage to extract its content.",
		InputSchema:                 inputSchema,
		OutputSchema:                nil,
		IsLongRunning:               false,
		RequireConfirmation:         false,
		RequireConfirmationProvider: func(input WebpageTopicInput) bool { return false },
	}, extractWebpageTopic)
}

func extractWebpageTopic(ctx toolx.Context, input WebpageTopicInput) (string, error) {
	httpClient := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, input.URL, nil)
	if err != nil {
		return "", err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch webpage: %s", resp.Status)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(raw), nil
}
