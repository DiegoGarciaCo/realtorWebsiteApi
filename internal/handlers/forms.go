package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type data struct {
	Price        int
	Interest     float64
	Years        int
	DownPayment  float64
	Payment      float64
	TotalPayment float64
	MonthlyPMI   float64
	Taxes        float64
	Insurance    float64
}

func (cfg *apiCfg) CalculateMortgage(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Price       string `json:"price"`
		Interest    string `json:"interest"`
		Years       string `json:"years"`
		DownPayment string `json:"downPayment"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		Number      string `json:"number"`
		Subscribed  bool   `json:"subscribed"`
	}
	type requestContact struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		PhoneNumbers []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"phone_numbers"`
		Emails []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"emails"`
		Source string `json:"source"`
		Status string `json:"status"`
	}
	type requestNote struct {
		ContactID string `json:"contact_id"`
		Note      string `json:"note"`
	}

	// Decode request
	formData := reqParams{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&formData)
	if err != nil {
		log.Printf("Error decoding request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not decode request", err)
		return
	}

	url := "https://server.soldbyghost.com/api/contacts"
	noteURL := "https://server.soldbyghost.com/api/notes"

	payload := requestContact{
		FirstName: formData.FirstName,
		LastName:  formData.LastName,
		PhoneNumbers: []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Number:    formData.Number,
				Type:      "mobile",
				IsPrimary: true,
			},
		},
		Emails: []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Email:     formData.Email,
				Type:      "personal",
				IsPrimary: true,
			},
		},
		Source: "Website",
		Status: "Lead",
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not marshal JSON", err)
		return
	}

	r, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadJSON))
	if err != nil {
		log.Printf("Error creating request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create request", err)
		return
	}

	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-API-Key", cfg.CRMAPIKey)

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Printf("Error sending request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not send request", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Error response from CRM: %s", resp.Status)
		respondWithError(w, resp.StatusCode, "Did not get successful response", err)
		return
	}

	// Extract contact ID from response
	var respData struct {
		ID string `json:"ID"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		log.Printf("Error decoding response: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not decode response", err)
		return
	}

	contactID := respData.ID

	// Create Note for Contact
	notePayload := requestNote{
		ContactID: contactID,
		Note: "Requested Mortgage Calculation with the following details:\n" +
			"Price: $" + formData.Price + "\n" +
			"Interest Rate: " + formData.Interest + "%\n" +
			"Loan Term: " + formData.Years + " years\n" +
			"Down Payment: " + formData.DownPayment + "%",
	}

	notePayloadJSON, err := json.Marshal(notePayload)
	if err != nil {
		log.Printf("Error marshalling note JSON: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not marshal note JSON", err)
		return
	}

	noteReq, err := http.NewRequest("POST", noteURL, bytes.NewBuffer(notePayloadJSON))
	if err != nil {
		log.Printf("Error creating note request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create note request", err)
		return
	}

	noteReq.Header.Set("Content-Type", "application/json")
	noteReq.Header.Set("X-API-Key", cfg.CRMAPIKey)

	noteResp, err := client.Do(noteReq)
	if err != nil {
		log.Printf("Error sending note request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not send note request", err)
		return
	}
	defer noteResp.Body.Close()

	if noteResp.StatusCode < 200 || noteResp.StatusCode >= 300 {
		log.Printf("Error response from CRM when creating note: %s", noteResp.Status)
		respondWithError(w, http.StatusInternalServerError, "Could not create note", err)
		return
	}

	// Send Email Response
	go func() {
		price, err := strconv.Atoi(formData.Price)
		if err != nil {
			log.Printf("Error converting Price to integer: %v", err)
			return
		}
		tax := float64(price) * 0.0211
		var pmi float64
		var monthlyPMI float64

		downPaymentPercent, err := strconv.ParseFloat(formData.DownPayment, 64)
		if err != nil {
			log.Printf("Error converting DownPayment to integer: %v", err)
			return
		}

		downPayment := float64(price) * (downPaymentPercent / 100)
		if float64(downPayment)/float64(price) < 0.2 {
			pmi = 0.0075
			price, err := strconv.Atoi(formData.Price)
			if err != nil {
				log.Printf("Error converting Price to integer: %v", err)
				return
			}
			monthlyPMI = (float64(price) * pmi) / 12
		} else {
			pmi = 0.0
			monthlyPMI = 0.0
		}

		interest, err := strconv.ParseFloat(formData.Interest, 64)
		if err != nil {
			log.Printf("Error converting Interest to float64: %v", err)
			return
		}

		years, err := strconv.Atoi(formData.Years)
		if err != nil {
			log.Printf("Error converting Years to integer: %v", err)
			return
		}
		payment := CalculateMortgagePayment(float64(price)-float64(downPayment), interest, years)
		totalPayment := payment + monthlyPMI + (tax / 12) + (2119 / 12)
		if err = cfg.SendMortgageCalculation(data{
			Price:        price,
			Interest:     interest * 100,
			Years:        years,
			DownPayment:  downPayment,
			Payment:      payment,
			TotalPayment: totalPayment,
			MonthlyPMI:   monthlyPMI,
			Taxes:        tax / 12,
			Insurance:    2119 / 12,
		}, formData.Email); err != nil {
			log.Printf("Error sending mortgage calculation email: %v", err)
			return
		}
	}()
	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) Estimate(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Name    string `json:"name"`
		Address string `json:"address"`
		City    string `json:"city"`
		State   string `json:"state"`
		Email   string `json:"email"`
		Number  string `json:"number"`
	}

	type requestContact struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		PhoneNumbers []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"phone_numbers"`
		Emails []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"emails"`
		Source  string `json:"source"`
		Status  string `json:"status"`
		Address string `json:"address"`
		City    string `json:"city"`
		State   string `json:"state"`
	}
	type requestNote struct {
		ContactID string `json:"contact_id"`
		Note      string `json:"note"`
	}

	var formData reqParams
	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(&formData); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	fisrtName := strings.Split(formData.Name, " ")[0]
	lastName := strings.Split(formData.Name, " ")[1]

	url := "https://server.soldbyghost.com/api/contacts"
	noteURL := "https://server.soldbyghost.com/api/notes"

	payload := requestContact{
		FirstName: fisrtName,
		LastName:  lastName,
		PhoneNumbers: []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Number:    formData.Number,
				Type:      "mobile",
				IsPrimary: true,
			},
		},
		Emails: []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Email:     formData.Email,
				Type:      "personal",
				IsPrimary: true,
			},
		},
		Source:  "Website",
		Status:  "Lead",
		Address: formData.Address,
		City:    formData.City,
		State:   formData.State,
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
	req.Header.Set("X-API-Key", cfg.CRMAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Extract contact ID from response
	var respData struct {
		ID string `json:"ID"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	contactID := respData.ID

	// Create Note for Contact
	notePayload := requestNote{
		ContactID: contactID,
		Note:      "Requested Property Estimate on the website.",
	}

	notePayloadJSON, err := json.Marshal(notePayload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	noteReq, err := http.NewRequest("POST", noteURL, bytes.NewBuffer(notePayloadJSON))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	noteReq.Header.Set("Content-Type", "application/json")
	noteReq.Header.Set("X-API-Key", cfg.CRMAPIKey)

	noteResp, err := client.Do(noteReq)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}
	defer noteResp.Body.Close()

	if noteResp.StatusCode < 200 || noteResp.StatusCode >= 300 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	respondWithJSON(w, http.StatusOK, nil)
}

func (cfg *apiCfg) SubmitForm(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		FirstName  string `json:"firstName"`
		LastName   string `json:"lastName"`
		Email      string `json:"email"`
		Number     string `json:"number"`
		Message    string `json:"message"`
		Subscribed bool   `json:"subscribed"`
	}
	type requestContact struct {
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
		PhoneNumbers []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"phone_numbers"`
		Emails []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		} `json:"emails"`
		Source string `json:"source"`
		Status string `json:"status"`
	}
	type requestNote struct {
		ContactID string `json:"contact_id"`
		Note      string `json:"note"`
	}

	formData := reqParams{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&formData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode request", err)
		return
	}
	url := "https://server.soldbyghost.com/api/contacts"
	notesURL := "https://server.soldbyghost.com/api/notes"

	payload := requestContact{
		FirstName: formData.FirstName,
		LastName:  formData.LastName,
		PhoneNumbers: []struct {
			Number    string `json:"number"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Number:    formData.Number,
				Type:      "mobile",
				IsPrimary: true,
			},
		},
		Emails: []struct {
			Email     string `json:"email"`
			Type      string `json:"type"`
			IsPrimary bool   `json:"is_primary"`
		}{
			{
				Email:     formData.Email,
				Type:      "personal",
				IsPrimary: true,
			},
		},
		Source: "Website",
		Status: "Lead",
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
	r.Header.Set("X-API-Key", cfg.CRMAPIKey)

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

	// Extract contact ID from response
	var respData struct {
		ID string `json:"ID"`
	}
	err = json.NewDecoder(resp.Body).Decode(&respData)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not decode response", err)
		return
	}

	contactID := respData.ID

	// Create Note for Contact
	notePayload := requestNote{
		ContactID: contactID,
		Note:      "Submitted a contact form with the following message:\n" + formData.Message,
	}

	notePayloadJSON, err := json.Marshal(notePayload)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not marshal note JSON", err)
		return
	}

	noteReq, err := http.NewRequest("POST", notesURL, bytes.NewBuffer(notePayloadJSON))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not create note request", err)
		return
	}

	noteReq.Header.Set("Content-Type", "application/json")
	noteReq.Header.Set("X-API-Key", cfg.CRMAPIKey)

	noteResp, err := client.Do(noteReq)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not send note request", err)
		return
	}
	defer noteResp.Body.Close()

	if noteResp.StatusCode != http.StatusOK {
		respondWithError(w, http.StatusInternalServerError, "Could not create note", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, reqParams{})
}
