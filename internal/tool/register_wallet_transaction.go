package tool

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/lopesgabriel/adk-go/internal/client/wallet"
	toolx "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type RegisterWalletTransactionInput struct {
	// Amount is the transaction value in cents (e.g. 15000 for R$150,00).
	Amount int `json:"amount"`
	// Description is a human-readable description of the transaction
	// (e.g. "Pix de João Silva - Jogo de panelas").
	Description string `json:"description"`
}

func NewRegisterWalletTransactionTool(client wallet.Client) (toolx.Tool, error) {
	inputSchema, err := jsonschema.For[RegisterWalletTransactionInput](nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate input schema for register_wallet_transaction: %w", err)
	}

	registerFn := func(ctx toolx.Context, input RegisterWalletTransactionInput) (string, error) {
		if input.Amount <= 0 {
			return "", fmt.Errorf("amount must be a positive integer representing cents (e.g. 15000 for R$150,00)")
		}
		if input.Description == "" {
			return "", fmt.Errorf("description is required")
		}

		transaction, err := client.RegisterTransaction(ctx, input.Amount, 100, input.Description)
		if err != nil {
			return "", fmt.Errorf("failed to register wallet transaction: %w", err)
		}

		result, err := json.Marshal(transaction)
		if err != nil {
			return "", fmt.Errorf("failed to marshal transaction result: %w", err)
		}

		return string(result), nil
	}

	return functiontool.New(functiontool.Config{
		Name: "register_wallet_transaction",
		Description: "Registers a deposit transaction in the housewarming wallet. " +
			"Use this tool after confirming a Pix payment matches a donation and updating the donation status to 'paid'. " +
			"The amount should be in cents (e.g. 15000 for R$150,00) and the description should identify the contributor and item.",
		InputSchema:         inputSchema,
		OutputSchema:        nil,
		IsLongRunning:       false,
		RequireConfirmation: false,
		RequireConfirmationProvider: func(input RegisterWalletTransactionInput) bool {
			return false
		},
	}, registerFn)
}
