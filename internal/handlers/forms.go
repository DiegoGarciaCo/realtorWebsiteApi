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

	// Follow up Boss API payload structures
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
		Source string `json:"source"`
		Type   string `json:"type"`
		Person Person `json:"person"`
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

	url := "https://api.followupboss.com/v1/events"

	payload := reqPayload{
		Source: cfg.System,
		Type:   "Property Inquiry",
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
	r.Header.Set("X-System", cfg.System)
	r.Header.Set("X-System-Key", cfg.SystemKey)
	r.SetBasicAuth(cfg.FUBKey, "")

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		log.Printf("Error sending request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not send request", err)
		return
	}
	defer resp.Body.Close()


	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("Error response from Follow Up Boss: %s", resp.Status)
		respondWithError(w, http.StatusInternalServerError, "Could not send request", err)
		return
	}

	// Create contact in Brevo
	contact := contact{
		Email: formData.Email,
		Attributes: attributes{
			FirstName: formData.FirstName,
			LastName:  formData.LastName,
			Sms:       formData.Number,
		},
		ListIDs: []int64{3},
		UpdateEnabled: true,
	}

	err = cfg.CreateContact(contact)
	if err != nil {
		log.Printf("Error creating contact in Brevo: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create contact in Brevo", err)
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
		}, formData.Email, cfg.AppPassword); err != nil {
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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Create contact in Brevo
	contact := contact{
		Email: formData.Email,
		Attributes: attributes{
			FirstName: fisrtName,
			LastName:  lastName,
			Sms:       formData.Number,
		},
		ListIDs: []int64{4},
		UpdateEnabled: true,
	}

	err = cfg.CreateContact(contact)
	if err != nil {
		log.Printf("Error creating contact in Brevo: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create contact in Brevo", err)
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


	// Create contact in Brevo
	contact := contact{
		Email: formData.Email,
		Attributes: attributes{
			FirstName: formData.FirstName,
			LastName:  formData.LastName,
			Sms:       formData.Number,
		},
		ListIDs: []int64{6},
		UpdateEnabled: true,
	}


	err = cfg.CreateContact(contact)
	if err != nil {
		log.Printf("Error creating contact in Brevo: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not create contact in Brevo", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, reqParams{})
}
