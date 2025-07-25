package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

type attributes struct {
	FirstName string `json:"FIRSTNAME"`
	LastName  string `json:"LASTNAME"`
	Sms	   string `json:"SMS"`
}

type contact struct {
	Email   string `json:"email"`
	Attributes attributes `json:"attributes"`
	ListIDs []int64 `json:"listIds"`
	UpdateEnabled bool `json:"updateEnabled"`
}
type ContactResponse struct {
	Email           string    `json:"email"`
	ID              int       `json:"id"`
	EmailBlacklisted bool     `json:"emailBlacklisted"`
	SmsBlacklisted   bool     `json:"smsBlacklisted"`
	CreatedAt       string    `json:"createdAt"`
	ModifiedAt      string    `json:"modifiedAt"`
	Attributes      struct {
		FirstName string `json:"FIRSTNAME"`
		LastName  string `json:"LASTNAME"`
		SMS       string `json:"SMS"`
	} `json:"attributes"`
	ListIDs    []int64         `json:"listIds"`
}

func (cfg *apiCfg) GetContactListIDs(email string) ([]int64, error) {
	endpoint := fmt.Sprintf("https://api.brevo.com/v3/contacts/%s", url.QueryEscape(email))

	// Prepare the request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", cfg.BrevoAPIKey)

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, fmt.Errorf("failed to get contact list IDs: %s", resp.Status)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, nil // Contact not found, return nil
	}

	var response ContactResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.ListIDs, nil
}


func (cfg *apiCfg) CreateContact(contact contact) error {
	endpoint := "https://api.brevo.com/v3/contacts"

	// Check if the contact already exists and get their list IDs
	listIDs, err := cfg.GetContactListIDs(contact.Email)
	if err != nil {
		return fmt.Errorf("error checking contact existence: %w", err)
	}

	contact.ListIDs = append(contact.ListIDs, listIDs...)
	contact.ListIDs = dedupe(contact.ListIDs)
	contact.Attributes.Sms = "+1" + contact.Attributes.Sms // Ensure SMS is in the correct format

	// Create or update the contact
	// Marshal the contact to JSON
	jsonData, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	// Prepare the request
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", cfg.BrevoAPIKey)

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read the response body for more details
		body, _ := io.ReadAll(resp.Body)
		if body != nil {
			return fmt.Errorf("failed to create contact: %s, response: %s", resp.Status, string(body))
		}
		return fmt.Errorf("failed to create contact: %s", resp.Status)
	}

	return nil
}
