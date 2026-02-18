package tool

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/lopesgabriel/adk-go/internal/client/casanova"
	toolx "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type ListDesiredItemsWithDonationsInput struct {
	// HousewarmingID is the unique identifier of the housewarming event.
	HousewarmingID string `json:"housewarming_id"`
}

func NewListDesiredItemsWithDonationsTool(client casanova.Client) (toolx.Tool, error) {
	inputSchema, err := jsonschema.For[ListDesiredItemsWithDonationsInput](nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate input schema for list_desired_items_with_donations: %w", err)
	}

	listFn := func(ctx toolx.Context, input ListDesiredItemsWithDonationsInput) (string, error) {
		if input.HousewarmingID == "" {
			return "", fmt.Errorf("housewarming_id is required")
		}

		items, err := client.ListDesiredItemsWithDonations(ctx, input.HousewarmingID)
		if err != nil {
			return "", fmt.Errorf("failed to list desired items with donations: %w", err)
		}

		result, err := json.Marshal(items)
		if err != nil {
			return "", fmt.Errorf("failed to marshal desired items with donations: %w", err)
		}

		return string(result), nil
	}

	return functiontool.New(functiontool.Config{
		Name: "list_desired_items_with_donations",
		Description: "Lists all desired items (gifts) for a housewarming event, with their associated donation entries embedded. " +
			"Each desired item has an id, name, description, suggested_price, and a list of donations. " +
			"Each donation has an id, donor_name, amount, status (pending/paid/canceled), and desired_item_id. " +
			"Use this tool to find a pending donation that matches a Pix transaction by comparing donor_name and amount.",
		InputSchema:         inputSchema,
		OutputSchema:        nil,
		IsLongRunning:       false,
		RequireConfirmation: false,
		RequireConfirmationProvider: func(input ListDesiredItemsWithDonationsInput) bool {
			return false
		},
	}, listFn)
}
