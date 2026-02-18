package tool

import (
	"encoding/json"
	"fmt"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/lopesgabriel/adk-go/internal/client/casanova"
	toolx "google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
)

type UpdateDonationStatusInput struct {
	// DonationID is the unique identifier of the donation to update.
	DonationID string `json:"donation_id"`
	// Status is the new status for the donation. Must be "paid".
	Status string `json:"status"`
}

func NewUpdateDonationStatusTool(client casanova.Client) (toolx.Tool, error) {
	inputSchema, err := jsonschema.For[UpdateDonationStatusInput](nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate input schema for update_donation_status: %w", err)
	}

	updateFn := func(ctx toolx.Context, input UpdateDonationStatusInput) (string, error) {
		if input.DonationID == "" {
			return "", fmt.Errorf("donation_id is required")
		}
		if input.Status == "" {
			return "", fmt.Errorf("status is required")
		}

		donation, err := client.UpdateDonationStatus(ctx, input.DonationID, input.Status)
		if err != nil {
			return "", fmt.Errorf("failed to update donation status: %w", err)
		}

		result, err := json.Marshal(donation)
		if err != nil {
			return "", fmt.Errorf("failed to marshal updated donation: %w", err)
		}

		return string(result), nil
	}

	return functiontool.New(functiontool.Config{
		Name: "update_donation_status",
		Description: "Updates the status of a donation entry in the housewarming system. " +
			"Requires the donation_id (from the donation object returned by list_desired_items_with_donations) " +
			"and the new status value. Set the status to 'paid' when a matching Pix transaction is confirmed.",
		InputSchema:         inputSchema,
		OutputSchema:        nil,
		IsLongRunning:       false,
		RequireConfirmation: false,
		RequireConfirmationProvider: func(input UpdateDonationStatusInput) bool {
			return false
		},
	}, updateFn)
}
