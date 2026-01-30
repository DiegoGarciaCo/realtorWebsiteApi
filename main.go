package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/DiegoGarciaCo/websitesAPI/internal/handlers"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keighl/postmark"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Println("PORT is not set, defaulting to 8080")
		port = "8080"
	}
	env := os.Getenv("ENV")
	if env == "" {
		env = "Production"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Println("DATABASE_URL is not set")
	}
	secret := os.Getenv("TOKEN_SECRET")
	if secret == "" {
		log.Println("TOKEN_SECRET is not set")
	}

	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		log.Println("S3_REGION is not set")
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Println("S3_BUCKET is not set")
	}
	betterAuthSecret := os.Getenv("BETTER_AUTH_SECRET")
	if betterAuthSecret == "" {
		log.Println("BETTER_AUTH_SECRET is not set")
	}
	postmarkServerToken := os.Getenv("POSTMARK_SERVER_TOKEN")
	if postmarkServerToken == "" {
		log.Println("POSTMARK_SERVER_TOKEN is not set")
	}
	fromEmail := os.Getenv("FROM_EMAIL_ADDRESS")
	if fromEmail == "" {
		log.Println("FROM_EMAIL_ADDRESS is not set")
	}
	crmAPIKey := os.Getenv("CRM_API_KEY")
	if crmAPIKey == "" {
		log.Println("CRM_API_KEY is not set")
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(s3Region))
	if err != nil {
		log.Println("Unable to load AWS SDK config:", err)
	}
	client := s3.NewFromConfig(awsCfg)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Println("Unable to connect to database:", err)
	}
	dbQueries := database.New(db)

	// ------------------------------------------------
	// Initialize Postmark client
	// ------------------------------------------------

	postmarkClient := postmark.Client{
		ServerToken: postmarkServerToken,
		BaseURL:     "https://api.postmarkapp.com",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
	EmailSecret := []byte(secret)

	apiCfg := handlers.NewConfig(port, secret, s3Bucket, s3Region, env, dbQueries, db, client, betterAuthSecret, &postmarkClient, EmailSecret, fromEmail, crmAPIKey)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://soldbyghost.com", "https://admin.soldbyghost.com"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowCredentials: true,
	})

	mux := http.NewServeMux()

	// Lead Submission
	mux.HandleFunc("POST /api/submit/form", apiCfg.SubmitForm)
	mux.HandleFunc("POST /api/calculator", apiCfg.CalculateMortgage)
	mux.HandleFunc("POST /api/estimate", apiCfg.Estimate)

	// Posts
	mux.HandleFunc("GET /api/posts/{slug}", apiCfg.PostBySlug)
	mux.HandleFunc("GET /api/posts/published", apiCfg.PublishedPost)
	mux.HandleFunc("GET /api/posts/category/{category}", apiCfg.GetPostsByCategory)
	mux.HandleFunc("GET /api/posts", apiCfg.AuthMiddleware(apiCfg.AllPosts))
	mux.HandleFunc("POST /api/posts/draft", apiCfg.AuthMiddleware(apiCfg.CreateDraftPost))
	mux.HandleFunc("POST /api/posts/publish/{id}", apiCfg.AuthMiddleware(apiCfg.PublishPost))
	mux.HandleFunc("PUT /api/posts/publish/{id}", apiCfg.AuthMiddleware(apiCfg.SaveAndPublishPost))
	mux.HandleFunc("POST /api/posts/publish", apiCfg.AuthMiddleware(apiCfg.PublishPost))
	mux.HandleFunc("POST /api/posts/thumbnail/{id}", apiCfg.AuthMiddleware(apiCfg.UploadThumnail))
	mux.HandleFunc("PUT /api/posts/thumbnail/{id}", apiCfg.AuthMiddleware(apiCfg.UpdateThumbnail))
	mux.HandleFunc("PUT /api/posts", apiCfg.AuthMiddleware(apiCfg.UpdatePost))
	mux.HandleFunc("DELETE /api/posts/delete/{id}", apiCfg.AuthMiddleware(apiCfg.DeletePost))

	handler := handlers.LoggerMiddleware(corsHandler.Handler(handlers.RecoveryMiddleware(mux)))

	srv := &http.Server{
		Addr:              ":" + apiCfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Print("Listening on port " + apiCfg.Port + "...")
	if err := srv.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("Server failed to start")
	}
}
