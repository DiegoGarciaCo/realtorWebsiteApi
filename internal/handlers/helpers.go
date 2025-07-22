package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gopkg.in/gomail.v2"
	"html/template"
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
	t, err := template.ParseFiles("emailBody.html")
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
