package handlers

import (
	"context"
	"io"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"log"
	"math"
	"net/http"
	"strings"
)

func respondWithError(w http.ResponseWriter, code int, msg string, err error) {
	if err != nil {
		log.Println(err)
	}
	if code > 499 {
		log.Printf("Responding with 5XX error: %s", msg)
	}
	type errorResponse struct {
		Error string `json:"error"`
	}
	respondWithJSON(w, code, errorResponse{
		Error: msg,
	})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")

	// Skip body for no-content statuses
	if code == http.StatusNoContent || code == http.StatusAccepted {
		w.WriteHeader(code)
		return
	}

	// Marshal and write payload for other statuses
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(code)
	if _, err := w.Write(data); err != nil {
		log.Printf("Error writing response: %s", err)
		// Can't call WriteHeader again, response already sent
	}
}

func mediaTypeToExt(mediaType string) string {
	parts := strings.Split(mediaType, "/")
	if len(parts) != 2 {
		return ".bin"
	}
	return "." + parts[1]
}

func getAssetPath(mediaType, folder string) string {
	base := make([]byte, 32)
	_, err := rand.Read(base)
	if err != nil {
		panic("failed to generate random bytes")
	}
	id := base64.RawURLEncoding.EncodeToString(base)

	ext := mediaTypeToExt(mediaType)
	return fmt.Sprintf("%s/%s%s", folder, id, ext)
}

func cleanupS3(cfg *apiCfg, ctx context.Context, keys []string) {
	if len(keys) == 0 {
		return
	}

	for _, key := range keys {
		_, err := cfg.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(cfg.S3Bucket),
			Key:    aws.String(key),
		})

		if err != nil {
			log.Printf("Failed to delete %s: %s", key, err)
		}
	}
}

func deleteFromS3(cfg *apiCfg, ctx context.Context, imageURL string) error {
	baseURL := "https://" + cfg.S3Bucket + ".s3." + cfg.S3Region + ".amazonaws.com/"
	key := strings.TrimPrefix(imageURL, baseURL)

	if key == imageURL {
		log.Printf("Invalid S3 URL format: %s", imageURL)
		return fmt.Errorf("Invalid S3 URL format: %s", imageURL)
	}

	_, err := cfg.S3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(cfg.S3Bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		log.Printf("Failed to delete %s: %s", key, err)
	}

	return nil
}

func CalculateMortgagePayment(P float64, annualRate float64, loanTerm int) float64 {
	// Convert annual interest rate to monthly
	monthlyRate := (annualRate / 100) / 12

	// Total number of payments (months)
	n := loanTerm * 12

	// Mortgage payment formula
	if monthlyRate == 0 {
		// In case of 0% interest rate (special case)
		return P / float64(n)
	}

	// Mortgage payment formula
	m := P * (monthlyRate * math.Pow(1+monthlyRate, float64(n))) / (math.Pow(1+monthlyRate, float64(n)) - 1)
	return math.Round(m*100) / 100
}

func (cfg *apiCfg) SendMortgageCalculation(data data, to, password string) error {
	// Request structures
	type To struct {
		Email string `json:"email"`
	}
	type bcc struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	type params struct {
		Price        int     `json:"Price"`
		Interest     float64 `json:"Interest"`
		Years        int     `json:"Years"`
		DownPayment  float64 `json:"DownPayment"`
		Payment      float64 `json:"Payment"`
		TotalPayment float64 `json:"TotalPayment"`
		MonthlyPMI   float64 `json:"MonthlyPMI"`
		Taxes        float64 `json:"Taxes"`
		Insurance    float64 `json:"Insurance"`
	}
	type brevoRequest struct {
		To        []To        `json:"to"`
		Bcc       []bcc         `json:"bcc"`
		TemplateID int          `json:"templateId"`
		Params    params       `json:"params"`
	}

	// Prepare the request 
	req := brevoRequest{
		To: []To{
			{
				Email: to,
			},
		},
		Bcc:       []bcc{
			{
				Email: "diegogarcia51916@gmail.com",
				Name:  "Diego Garcia",
			},
		},
		TemplateID: 1,
		Params: params{
			Price:        data.Price,
			Interest:     data.Interest,
			Years:        data.Years,
			DownPayment:  data.DownPayment,
			Payment:      data.Payment,
			TotalPayment: data.TotalPayment,
			MonthlyPMI:   data.MonthlyPMI,
			Taxes:        data.Taxes,
			Insurance:    data.Insurance,
		},
	}

	// Marshal the request to JSON
	jsonData, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshalling JSON for email request: %v", err)
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	// Send the request to the email service
	client := &http.Client{}
	reqEmail, err := http.NewRequest("POST", "https://api.brevo.com/v3/smtp/email", strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("Error creating HTTP request for email: %v", err)
		return fmt.Errorf("failed to create HTTP request for email: %w", err)
	}
	reqEmail.Header.Set("Content-Type", "application/json")
	reqEmail.Header.Set("api-key", cfg.BrevoAPIKey)

	
	resp, err := client.Do(reqEmail)
	if err != nil {
		log.Printf("Error sending email request: %v", err)
		return fmt.Errorf("failed to send email request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Email request failed with status: %s", resp.Status)
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("email request failed with status %s: %s", resp.Status, body)
	}
	log.Println("Mortgage calculation email sent successfully")

	return nil
}
