package casanova

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client defines the interface for interacting with the casa-nova housewarming API.
type Client interface {
	// ListDesiredItemsWithDonations returns all desired items with their associated
	// donations for a given housewarming.
	ListDesiredItemsWithDonations(ctx context.Context, housewarmingID string) ([]DesiredItemWithDonations, error)

	// UpdateDonationStatus updates the status of a specific donation entry.
	// The status should be a valid DonationStatus value (e.g. "paid").
	UpdateDonationStatus(ctx context.Context, donationID string, newStatus string) (*Donation, error)
}

type httpClient struct {
	baseURL      string
	serviceToken string
	client       *http.Client
}

// NewClient creates a new casa-nova HTTP client.
// baseURL should point to the casa-nova backend (e.g. "http://localhost:8080").
// serviceToken is the SERVICE_TOKEN used for service-to-service authentication
// on protected endpoints (e.g. PATCH /donations/{id}/status).
func NewClient(baseURL, serviceToken string) Client {
	return &httpClient{
		baseURL:      baseURL,
		serviceToken: serviceToken,
		client:       &http.Client{},
	}
}

func (c *httpClient) ListDesiredItemsWithDonations(ctx context.Context, housewarmingID string) ([]DesiredItemWithDonations, error) {
	url := fmt.Sprintf("%s/housewarmings/%s/desired-items-with-donations", c.baseURL, housewarmingID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create list desired items request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list desired items with donations: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("list desired items with donations failed with status %d: %s", resp.StatusCode, string(body))
	}

	var items []DesiredItemWithDonations
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode desired items with donations response: %w", err)
	}

	return items, nil
}

func (c *httpClient) UpdateDonationStatus(ctx context.Context, donationID string, newStatus string) (*Donation, error) {
	url := fmt.Sprintf("%s/donations/%s/status", c.baseURL, donationID)

	payload := map[string]string{"new_status": newStatus}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create update donation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.serviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.serviceToken)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update donation status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("update donation status failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var donation Donation
	if err := json.NewDecoder(resp.Body).Decode(&donation); err != nil {
		return nil, fmt.Errorf("failed to decode updated donation: %w", err)
	}

	return &donation, nil
}
