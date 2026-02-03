package main

import (
	"context"
	"log"
	"os"

	"github.com/lopesgabriel/adk-go/internal/client/ollama"
	"github.com/lopesgabriel/adk-go/internal/model"
	"github.com/lopesgabriel/adk-go/internal/tool"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	toolx "google.golang.org/adk/tool"
)

func main() {
	ctx := context.Background()

	ollamaClient := ollama.NewOllamaHTTPClient("http://localhost:11434")
	model := model.NewOllamaModelAdapter("qwen3:8b", ollamaClient)

	webpageTopicTool, err := tool.NewWebpageTopicTool()
	if err != nil {
		log.Fatalf("Failed to create webpage topic tool: %v", err)
	}

	instruction := "You are a helpful assistant that extracts the topic of a webpage.\n"
	instruction += "You should always resume a webpage content to a single topic keyword or phrase.\n"
	instruction += "When given a URL, use the get_webpage tool to access the webpage.\n"
	instruction += "Once you have the page content, extract the topic and return it."

	timeAgent, err := llmagent.New(llmagent.Config{
		Name:        "webpage_topic_agent",
		Model:       model,
		Description: "Extracts the topic of a webpage.",
		Instruction: instruction,
		Tools:       []toolx.Tool{webpageTopicTool},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(timeAgent),
	}

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}
