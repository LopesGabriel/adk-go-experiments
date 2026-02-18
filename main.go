package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/lopesgabriel/adk-go/internal/client/casanova"
	"github.com/lopesgabriel/adk-go/internal/client/wallet"
	"github.com/lopesgabriel/adk-go/internal/tool"
	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/cmd/launcher"
	"google.golang.org/adk/cmd/launcher/full"
	"google.golang.org/adk/model/gemini"
	toolx "google.golang.org/adk/tool"
	"google.golang.org/genai"
)

const agentInstruction = `You are a specialized agent for the "Chá de Casa Nova" (Housewarming) system.
Your primary job is to process email content and identify Brazilian Pix transaction notifications.

## How to handle email content

You will receive the raw content of an email. The email may be in text/html format with a lot of
HTML tags and CSS styling. You MUST ignore all HTML markup, CSS styles, and formatting — focus
exclusively on the textual content within the email.

## Identifying Pix transactions

A Pix transaction notification email typically contains:
- The name of the person who sent the Pix transfer
- The amount transferred (in BRL - Brazilian Real, e.g. R$ 150,00)
- The date and time of the transaction

Extract these three pieces of information from the email content. If the email is NOT a Pix
transaction notification, respond explaining that the email does not appear to be a Pix notification.

## Matching with donations

After extracting the Pix transaction information, use the list_desired_items_with_donations tool
(passing the housewarming_id provided in your configuration) to retrieve all desired items with
their associated donation entries.

Each desired item has a name, description, and suggested_price. Each item may have zero or more
donations. Each donation has a donor_name, amount, and status (pending, paid, or canceled).

Try to match the Pix transaction with a **pending** donation by comparing:
1. The donor_name from the donation with the sender name from the Pix transaction
2. The amount from the donation with the transferred amount

Name matching should be flexible — consider partial matches, nicknames, or variations
(e.g. "João Silva" should match "JOAO DA SILVA"). Amount matching should allow for small
discrepancies. Maximum allowed discrepancy is 250 BRL (e.g. a Pix of R$ 250,00 could match a donation of R$ 150,00).

If multiple matches are found, choose the best match based on name and amount similarity.

## Updating the donation

If you find a matching donation entry with status "pending", use the update_donation_status tool
to change its status to "paid". Pass the donation's id and status "paid".

If no match is found, proceed without updating any donation status. In this case, you should
add a note in the description field of the wallet transaction (see next section) indicating that
no matching donation was found.

## Registering the wallet transaction

After processing the donation (whether matched or not), you MUST register a deposit transaction
in the housewarming wallet using the register_wallet_transaction tool.
- The amount must be in cents (e.g. R$150,00 = 15000).
- The description should identify the donor and, if matched, the desired item name
  (e.g. "Pix de João Silva - Jogo de panelas").
- If no matching donation was found, note that in the description
  (e.g. "Pix de João Silva - sem doação correspondente encontrada").

## Summary

Always provide a clear summary of what you did: the extracted Pix info, whether a match was found,
the donation status update, and the wallet transaction registration.`

func main() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Ollama LLM
	// ollamaBaseURL := envOrDefault("OLLAMA_BASE_URL", "http://localhost:11434")
	// ollamaModel := envOrDefault("OLLAMA_MODEL", "qwen3:8b")

	// ollamaClient := ollama.NewOllamaHTTPClient(ollamaBaseURL)
	// llm := model.NewOllamaModelAdapter(ollamaModel, ollamaClient)

	// Gemini LLM
	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	// Casa-nova API client
	casanovaBaseURL := envOrDefault("CASANOVA_BASE_URL", "http://localhost:8080")
	casanovaServiceToken := os.Getenv("CASANOVA_SERVICE_TOKEN")
	casanovaHousewarmingID := os.Getenv("CASANOVA_HOUSEWARMING_ID")

	if casanovaServiceToken == "" {
		log.Fatalf("CASANOVA_SERVICE_TOKEN is required")
	}
	if casanovaHousewarmingID == "" {
		log.Fatalf("CASANOVA_HOUSEWARMING_ID is required")
	}

	casanovaClient := casanova.NewClient(casanovaBaseURL, casanovaServiceToken)

	// Wallet service client
	walletAPIBaseURL := envOrDefault("WALLET_API_BASE_URL", "http://localhost:8083")
	walletAuthBaseURL := envOrDefault("WALLET_AUTH_BASE_URL", "http://localhost:8081")
	walletEmail := envOrDefault("WALLET_EMAIL", "")
	walletPassword := envOrDefault("WALLET_PASSWORD", "")
	walletName := envOrDefault("WALLET_NAME", "Chá de casa nova")

	walletClient, err := wallet.NewClient(ctx, walletAPIBaseURL, walletAuthBaseURL, walletEmail, walletPassword, walletName)
	if err != nil {
		log.Fatalf("Failed to create wallet client: %v", err)
	}

	// Tools
	listDesiredItemsTool, err := tool.NewListDesiredItemsWithDonationsTool(casanovaClient)
	if err != nil {
		log.Fatalf("Failed to create list_desired_items_with_donations tool: %v", err)
	}

	updateDonationStatusTool, err := tool.NewUpdateDonationStatusTool(casanovaClient)
	if err != nil {
		log.Fatalf("Failed to create update_donation_status tool: %v", err)
	}

	registerWalletTransactionTool, err := tool.NewRegisterWalletTransactionTool(walletClient)
	if err != nil {
		log.Fatalf("Failed to create register_wallet_transaction tool: %v", err)
	}

	// Agent
	instruction := agentInstruction + fmt.Sprintf("\n\n## Configuration\n\nThe housewarming_id for this session is: %s", casanovaHousewarmingID)

	housewarmingAgent, err := llmagent.New(llmagent.Config{
		Name:        "housewarming_pix_agent",
		Model:       model,
		Description: "Processes email content to identify Brazilian Pix transactions and match them with housewarming donation entries.",
		Instruction: instruction,
		Tools:       []toolx.Tool{listDesiredItemsTool, updateDonationStatusTool, registerWalletTransactionTool},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	config := &launcher.Config{
		AgentLoader: agent.NewSingleLoader(housewarmingAgent),
	}

	l := full.NewLauncher()
	if err = l.Execute(ctx, config, os.Args[1:]); err != nil {
		log.Fatalf("Run failed: %v\n\n%s", err, l.CommandLineSyntax())
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
