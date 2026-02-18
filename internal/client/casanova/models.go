package casanova

import "time"

// DesiredItem represents an item the organizer wishes to receive as a gift.
type DesiredItem struct {
	ID             string     `json:"id"`
	HouseWarmingID string     `json:"housewarming_id"`
	Name           string     `json:"name"`
	Description    string     `json:"description"`
	SuggestedPrice float64    `json:"suggested_price"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at"`
}

// Donation represents a user's commitment to gift a desired item.
type Donation struct {
	ID            string     `json:"id"`
	DesiredItemID string     `json:"desired_item_id"`
	DonorID       string     `json:"donor_id"`
	DonorName     string     `json:"donor_name"`
	Amount        float64    `json:"amount"`
	Status        string     `json:"status"` // "pending", "paid", "canceled"
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}

// DesiredItemWithDonations combines a DesiredItem with its associated Donations.
type DesiredItemWithDonations struct {
	DesiredItem
	Donations []Donation `json:"donations"`
}
