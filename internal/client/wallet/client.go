package wallet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client defines the interface for interacting with the Tellawl Wallet service.
type Client interface {
	// RegisterTransaction creates a deposit transaction in the housewarming wallet.
	// amount is the integer value (e.g. 15000 for R$150,00), offset is the divisor (typically 100).
	RegisterTransaction(ctx context.Context, amount int, offset int, description string) (*Transaction, error)
}

type httpClient struct {
	baseURL    string
	walletName string
	email      string
	password   string

	client      *http.Client
	credentials *Credentials
	walletID    string
}

// NewClient creates a new Wallet HTTP client. It authenticates with the wallet service
// and resolves the wallet ID for the given walletName on first use.
func NewClient(ctx context.Context, baseURL, email, password, walletName string) (Client, error) {
	c := &httpClient{
		baseURL:    baseURL,
		walletName: walletName,
		email:      email,
		password:   password,
		client:     &http.Client{},
	}

	if err := c.signIn(ctx); err != nil {
		return nil, fmt.Errorf("wallet client: failed to sign in: %w", err)
	}

	if err := c.resolveWalletID(ctx); err != nil {
		return nil, fmt.Errorf("wallet client: failed to resolve wallet ID: %w", err)
	}

	return c, nil
}

func (c *httpClient) signIn(ctx context.Context) error {
	payload := map[string]string{
		"email":    c.email,
		"password": c.password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sign-in payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/public/signin", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create sign-in request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("sign-in request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sign-in returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var creds Credentials
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		return fmt.Errorf("failed to decode sign-in response: %w", err)
	}
	c.credentials = &creds
	return nil
}

func (c *httpClient) refreshToken(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/public/refresh-token", nil)
	if err != nil {
		return fmt.Errorf("failed to create refresh-token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.credentials.RefreshToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("refresh-token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("refresh-token returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var creds Credentials
	if err := json.NewDecoder(resp.Body).Decode(&creds); err != nil {
		return fmt.Errorf("failed to decode refresh-token response: %w", err)
	}
	c.credentials = &creds
	return nil
}

func (c *httpClient) resolveWalletID(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/wallets", nil)
	if err != nil {
		return fmt.Errorf("failed to create list wallets request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.credentials.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("list wallets request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.refreshToken(ctx); err != nil {
			return fmt.Errorf("failed to refresh token during wallet resolution: %w", err)
		}
		return c.resolveWalletID(ctx)
	}

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("list wallets returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var wallets []Wallet
	if err := json.NewDecoder(resp.Body).Decode(&wallets); err != nil {
		return fmt.Errorf("failed to decode wallets response: %w", err)
	}

	for _, w := range wallets {
		if w.Name == c.walletName {
			c.walletID = w.ID
			return nil
		}
	}

	return fmt.Errorf("wallet %q not found", c.walletName)
}

func (c *httpClient) doAuthenticatedRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.credentials.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Retry once on 401 after refreshing the token
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.refreshToken(ctx); err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.credentials.Token)

		return c.client.Do(req)
	}

	return resp, nil
}

func (c *httpClient) RegisterTransaction(ctx context.Context, amount int, offset int, description string) (*Transaction, error) {
	if offset == 0 {
		offset = 100
	}

	url := fmt.Sprintf("%s/wallets/%s/transactions", c.baseURL, c.walletID)

	payload := map[string]any{
		"amount":           amount,
		"offset":           offset,
		"transaction_type": "deposit",
		"description":      description,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal transaction payload: %w", err)
	}

	resp, err := c.doAuthenticatedRequest(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("register transaction request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("register transaction returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var transaction Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transaction); err != nil {
		return nil, fmt.Errorf("failed to decode transaction response: %w", err)
	}

	return &transaction, nil
}
