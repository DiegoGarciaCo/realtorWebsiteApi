package main

import (
	"bytes"
	"encoding/json"
	"html/template"
	"log"
	"math"
	"net/http"

	"gopkg.in/gomail.v2"
)

type data struct {
	Price        int
	Interest     float64
	Years        int
	DownPayment  int
	Payment      float64
	TotalPayment float64
	MonthlyPMI   float64
	Taxes        float64
	Insurance    float64
}

func (cfg *apiCfg) calculateMortgage(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Price       int     `json:"Price"`
		Interest    float64 `json:"interest"`
		Years       int     `json:"years"`
		DownPayment int     `json:"downPayment"`
		FirstName   string  `json:"firstName"`
		LastName    string  `json:"lastName"`
		Email       string  `json:"email"`
		Number      string  `json:"number"`
		Subscribed  bool    `json:"subscribed"`
	}

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

	// Send Email Response
	go func() {
		tax := float64(formData.Price) * 0.0211
		var pmi float64
		var monthlyPMI float64

		if float64(formData.DownPayment)/float64(formData.Price) < 0.2 {
			pmi = 0.0075
			monthlyPMI = (float64(formData.Price) * pmi) / 12
		} else {
			pmi = 0.0
			monthlyPMI = 0.0
		}

		payment := CalculateMortgagePayment(float64(formData.Price)-float64(formData.DownPayment), formData.Interest, formData.Years)
		totalPayment := payment + monthlyPMI + (tax / 12) + (2119 / 12)
		if err = SendEmail(data{
			Price:        formData.Price,
			Interest:     formData.Interest * 100,
			Years:        formData.Years,
			DownPayment:  formData.DownPayment,
			Payment:      payment,
			TotalPayment: totalPayment,
			MonthlyPMI:   monthlyPMI,
			Taxes:        tax / 12,
			Insurance:    2119 / 12,
		}, formData.Email, cfg.appPassword); err != nil {
			return
		}
	}()
	respondWithJSON(w, http.StatusNoContent, nil)
}

func CalculateMortgagePayment(P float64, annualRate float64, loanTerm int) float64 {
	// Convert annual interest rate to monthly
	monthlyRate := annualRate / 12

	// Total number of payments (months)
	n := loanTerm * 12

	// Mortgage payment formula
	if monthlyRate == 0 {
		// In case of 0% interest rate (special case)
		return P / float64(n)
	}

	// Mortgage payment formula
	m := P * (monthlyRate * math.Pow(1+monthlyRate, float64(n))) / (math.Pow(1+monthlyRate, float64(n)) - 1)
	return m
}

func SendEmail(data data, to, password string) error {
	// Creat email body from template
	var emailBody bytes.Buffer
	t, err := template.ParseFiles("./emailBody.html")
	if err != nil {
		return err
	}
	err = t.Execute(&emailBody, data)
	if err != nil {
		log.Print(err)
	}

	// create email
	m := gomail.NewMessage()
	m.SetHeader("From", "diego@stonebrgrealy.com")
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Your Mortgage Calculation")
	m.SetBody("text/html", emailBody.String())

	d := gomail.NewDialer("smtp.gmail.com", 587, "diego@stonebrgrealty.com", password)

	// Send the email
	if err := d.DialAndSend(m); err != nil {
		return err
	}

	return nil
}
