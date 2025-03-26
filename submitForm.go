package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

func (cfg *apiCfg) submitForm(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		FirstName  string `json:"firstName"`
		LastName   string `json:"lastName"`
		Email      string `json:"email"`
		Number     string `json:"number"`
		Message    string `json:"message"`
		Subscribed bool   `json:"subscribed"`
	}

	// Person represents the lead’s details within the payload
	type Email struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}

	type Phone struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}
	type Person struct {
		FirstName string  `json:"firstName"`
		LastName  string  `json:"lastName"`
		Emails    []Email `json:"emails"`
		Phones    []Phone `json:"phones"`
	}
	type reqPayload struct {
		Source  string `json:"source"`
		Type    string `json:"type"`
		Message string `json:"message"`
		Person  Person `json:"person"`
	}

	formData := reqParams{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&formData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode request", err)
		return
	}
	url := "https://api.followupboss.com/v1/events"

	payload := reqPayload{
		Source:  cfg.System,
		Type:    "General Inquiry",
		Message: formData.Message,
		Person: Person{
			FirstName: formData.FirstName,
			LastName:  formData.LastName,
			Emails: []Email{
				{
					Value: formData.Email,
					Type:  "Personal",
				},
			},
			Phones: []Phone{
				{
					Value: formData.Number,
					Type:  "Mobile",
				},
			},
		},
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal JSON", err)
		return
	}

	r, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create request", err)
		return
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-System", cfg.System)
	r.Header.Set("X-System-Key", cfg.SystemKey)
	r.SetBasicAuth(cfg.FUBKey, "")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not send request", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respondWithError(w, http.StatusInternalServerError, "Could not send request", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, reqParams{})
}
