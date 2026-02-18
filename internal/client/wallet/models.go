package wallet

import "time"

// Credentials holds the authentication tokens for the wallet service.
type Credentials struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Monetary represents a fixed-point monetary value (real value = Value / Offset).
type Monetary struct {
	Value  int `json:"value"`
	Offset int `json:"offset"`
}

// Member represents a user in the wallet service.
type Member struct {
	ID        string     `json:"id"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// Transaction represents a wallet transaction (deposit or withdraw).
type Transaction struct {
	ID              string     `json:"id"`
	Amount          Monetary   `json:"amount"`
	CreatedBy       Member     `json:"created_by"`
	TransactionType string     `json:"transaction_type"`
	Description     string     `json:"description"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       *time.Time `json:"updated_at"`
}

// Wallet represents a wallet in the wallet service.
type Wallet struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	CreatorID string     `json:"creator_id"`
	Balance   Monetary   `json:"balance"`
	Members   []Member   `json:"members"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}
