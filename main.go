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
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT is not set")
	}
	env := os.Getenv("ENV")
	if env == "" {
		env = "Production"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}
	secret := os.Getenv("TOKEN_SECRET")
	if secret == "" {
		log.Fatal("TOKEN_SECRET is not set")
	}
	appPassword := os.Getenv("APP_PASSWORD")
	if appPassword == "" {
		log.Fatal("APP_PASSWORD is not set")
	}
	FUBKey := os.Getenv("FUB_API_KEY")
	if FUBKey == "" {
		log.Fatal("FUB_API_KEY is not set")
	}
	system := os.Getenv("X_SYSTEM")
	if system == "" {
		log.Fatal("X_SYSTEM is not set")
	}
	systemKey := os.Getenv("X_SYSTEM_KEY")
	if systemKey == "" {
		log.Fatal("X_SYSTEM_KEY is not set")
	}

	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		log.Fatal("S3_REGION is not set")
	}
	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET is not set")
	}
	brevoAPIKey := os.Getenv("BREVO_API_KEY")
	if brevoAPIKey == "" {
		log.Fatal("BREVO_API_KEY is not set")
	}

	awsCfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(s3Region))
	if err != nil {
		log.Fatal(err)
	}
	client := s3.NewFromConfig(awsCfg)

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Unable to connect to database ", err)
	}
	dbQueries := database.New(db)

	apiCfg := handlers.NewConfig(port, secret, appPassword, FUBKey, system, systemKey, s3Bucket, s3Region, brevoAPIKey, env, dbQueries, db, client)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://soldbyghost.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowCredentials: true,
	})

	mux := http.NewServeMux()

	// Lead Submission
	mux.HandleFunc("POST /api/submit/form", apiCfg.SubmitForm)
	mux.HandleFunc("POST /api/calculator", apiCfg.CalculateMortgage)
	mux.HandleFunc("POST /api/estimate", apiCfg.Estimate)

	// Auth
	mux.HandleFunc("POST /api/auth/login", apiCfg.Login)
	mux.HandleFunc("POST /api/auth/logout", apiCfg.Logout)
	mux.HandleFunc("POST /api/auth/refresh", apiCfg.RefreshToken)
	mux.HandleFunc("POST /api/auth/validate", apiCfg.ValidateJWT)

	// Posts
	mux.HandleFunc("GET /api/posts/{slug}", apiCfg.PostBySlug)
	mux.HandleFunc("GET /api/posts/published", apiCfg.PublishedPost)
	mux.HandleFunc("GET /api/posts", apiCfg.AllPosts)
	mux.HandleFunc("POST /api/posts/draft", apiCfg.CreateDraftPost)
	mux.HandleFunc("POST /api/posts/publish/{id}", apiCfg.PublishPost)
	mux.HandleFunc("PUT /api/posts/publish/{id}", apiCfg.SaveAndPublishPost)
	mux.HandleFunc("POST /api/posts/publish", apiCfg.PublishPost)
	mux.HandleFunc("POST /api/posts/thumbnail/{id}", apiCfg.UploadThumnail)
	mux.HandleFunc("PUT /api/posts/thumbnail/{id}", apiCfg.UpdateThumbnail)
	mux.HandleFunc("PUT /api/posts", apiCfg.UpdatePost)
	mux.HandleFunc("DELETE /api/posts/delete/{id}", apiCfg.DeletePost)

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
