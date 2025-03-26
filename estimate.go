package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

func (cfg *apiCfg) Estimate(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		City    string `json:"city"`
		State   string `json:"state"`
		Email   string `json:"email"`
		Number  string `json:"number"`
	}

	type Email struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}

	type Phone struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}
	type Address struct {
		Type   string `json:"type"`
		Street string `json:"street"`
		City   string `json:"city"`
		State  string `json:"state"`
	}
	type Person struct {
		FirstName string    `json:"firstName"`
		LastName  string    `json:"lastName"`
		Emails    []Email   `json:"emails"`
		Phones    []Phone   `json:"phones"`
		Addresses []Address `json:"addresses"`
	}
	type reqPayload struct {
		Source string `json:"source"`
		Type   string `json:"type"`
		Person Person `json:"person"`
	}

	var formData reqParams
	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&formData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	fisrtName := strings.Split(formData.Name, " ")[0]
	lastName := strings.Split(formData.Name, " ")[1]

	url := "https://api.followupboss.com/v1/events"

	payload := reqPayload{
		Source: "Realtor Website",
		Type:   "Seller Inquiry",
		Person: Person{
			FirstName: fisrtName,
			LastName:  lastName,
			Emails: []Email{
				{
					Value: formData.Email,
					Type:  "personal",
				},
			},
			Phones: []Phone{
				{
					Value: formData.Number,
					Type:  "personal",
				},
			},
			Addresses: []Address{
				{
					Type:   "home",
					Street: formData.Address,
					City:   formData.City,
					State:  formData.State,
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	req, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-System", cfg.System)
	req.Header.Set("X-System-Key", cfg.SystemKey)
	req.SetBasicAuth(cfg.FUBKey, "")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	respondWithJSON(w, http.StatusOK, nil)
}
