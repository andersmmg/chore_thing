package grocy

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// GrocyClient is a struct to hold the base URL and API key for the Grocy API
type GrocyClient struct {
	BaseURL string
	APIKey  string
}

// NewGrocyClient creates a new instance of GrocyClient
func NewGrocyClient(baseURL, apiKey string) *GrocyClient {
	return &GrocyClient{
		BaseURL: baseURL,
		APIKey:  apiKey,
	}
}

// GetChores retrieves the list of chores from the Grocy API
func (c *GrocyClient) GetChores() ([]Chore, error) {
	url := fmt.Sprintf("%s/chores", c.BaseURL)
	fmt.Println("Requesting URL:", url) // Log the URL for debugging

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("GROCY-API-KEY", c.APIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get chores: %s", resp.Status)
	}

	var chores []Chore
	if err := json.NewDecoder(resp.Body).Decode(&chores); err != nil {
		return nil, err
	}

	return chores, nil
}

// GetUsers retrieves the list of users from the Grocy API
func (c *GrocyClient) GetUsers() ([]User, error) {
	url := fmt.Sprintf("%s/users", c.BaseURL)
	fmt.Println("Requesting URL:", url) // Log the URL for debugging

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("GROCY-API-KEY", c.APIKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get users: %s", resp.Status)
	}

	var users []User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, err
	}

	return users, nil
}

// Chore represents a chore in Grocy
type Chore struct {
	ChoreID                       int     `json:"chore_id"`
	ChoreName                     string  `json:"chore_name"`
	LastTrackedTime               string  `json:"last_tracked_time"`
	TrackDateOnly                 BoolInt `json:"track_date_only"` // Use custom type
	NextEstimatedExecutionTime    string  `json:"next_estimated_execution_time"`
	NextExecutionAssignedToUserID int     `json:"next_execution_assigned_to_user_id"`
	IsRescheduled                 BoolInt `json:"is_rescheduled"` // Use custom type
	IsReassigned                  BoolInt `json:"is_reassigned"`  // Use custom type
	NextExecutionAssignedUser     User    `json:"next_execution_assigned_user"`
}

// User represents a user in Grocy
type User struct {
	ID                  int    `json:"id"`
	Username            string `json:"username"`
	FirstName           string `json:"first_name"`
	LastName            string `json:"last_name"`
	DisplayName         string `json:"display_name"`
	PictureFileName     string `json:"picture_file_name"`
	RowCreatedTimestamp string `json:"row_created_timestamp"`
}

// BoolInt is a custom type to handle boolean values represented as integers
type BoolInt bool

// UnmarshalJSON implements the json.Unmarshaler interface for BoolInt
func (b *BoolInt) UnmarshalJSON(data []byte) error {
	var num int
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*b = BoolInt(num != 0) // Convert 0 to false, any other number to true
	return nil
}
