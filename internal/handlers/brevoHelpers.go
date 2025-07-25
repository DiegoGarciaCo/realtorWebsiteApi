package handlers

import (
	"bytes"
	"io"
	"encoding/json"
	"fmt"
	"net/http"
)

type attributes struct {
	FirstName string `json:"FNAME"`
	LastName  string `json:"LNAME"`
	Sms	   string `json:"SMS"`
}

type contact struct {
	Email   string `json:"email"`
	Attributes attributes `json:"attributes"`
	ListIDs []int64 `json:"listIds"`
}



func (cfg *apiCfg) CreateContact(contact contact) error {
	endpoint := "https://api.brevo.com/v3/contacts"

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

	if resp.StatusCode != http.StatusCreated {
		// Read the response body for more details
		body, _ := io.ReadAll(resp.Body)
		if body != nil {
			return fmt.Errorf("failed to create contact: %s, response: %s", resp.Status, string(body))
		}
		return fmt.Errorf("failed to create contact: %s", resp.Status)
	}

	return nil
}
