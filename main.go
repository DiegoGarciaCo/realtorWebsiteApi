package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"
)

type apiCfg struct {
	port        string
	secret      string
	DB          *database.Queries
	SQLDB       *sql.DB
	appPassword string
	FUBKey      string
	System      string
	SystemKey   string
	S3Client    *s3.Client
	S3Bucket    string
	S3Region    string
	Env         string
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
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

	apiCfg := apiCfg{
		port:        ":" + port,
		secret:      secret,
		DB:          dbQueries,
		SQLDB:       db,
		appPassword: appPassword,
		FUBKey:      FUBKey,
		System:      system,
		SystemKey:   systemKey,
		S3Client:    client,
		S3Bucket:    s3Bucket,
		S3Region:    s3Region,
		Env:         env,
	}

	corsHandler := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowCredentials: true,
	})

	mux := http.NewServeMux()

	// Lead Submission
	mux.HandleFunc("POST /api/submit/form", apiCfg.submitForm)
	mux.HandleFunc("POST /api/rateLeads", apiCfg.calculateMortgage)
	mux.HandleFunc("POST /api/estimate", apiCfg.Estimate)

	// Auth
	mux.HandleFunc("POST /api/auth/login", apiCfg.Login)
	mux.HandleFunc("POST /api/auth/logout", apiCfg.Logout)
	mux.HandleFunc("POST /api/auth/refresh", apiCfg.RefreshToken)
	mux.HandleFunc("POST /api/auth/validate", apiCfg.ValidateJWT)

	// Posts
	mux.HandleFunc("GET /api/posts/{slug}", apiCfg.postBySlug)
	mux.HandleFunc("GET /api/posts/published", apiCfg.PublishedPost)
	mux.HandleFunc("GET /api/posts", apiCfg.AuthMiddleware(apiCfg.allPosts))
	mux.HandleFunc("POST /api/posts/draft", apiCfg.AuthMiddleware(apiCfg.createDraftPost))
	mux.HandleFunc("POST /api/posts/publish/{id}", apiCfg.AuthMiddleware(apiCfg.PublishPost))
	mux.HandleFunc("POST /api/posts/thumbnail/{id}", apiCfg.AuthMiddleware(apiCfg.uploadThumnail))
	mux.HandleFunc("PUT /api/posts/update/{id}", apiCfg.AuthMiddleware(apiCfg.updatePost))
	mux.HandleFunc("DELETE /api/posts/delete/{id}", apiCfg.AuthMiddleware(apiCfg.deletePost))

	handler := LoggerMiddleware(corsHandler.Handler(RecoveryMiddleware(mux)))

	srv := &http.Server{
		Addr:              apiCfg.port,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Print("Listening on port " + apiCfg.port + " ...")
	if err := srv.ListenAndServe(); err != nil {
		logrus.WithError(err).Fatal("Server failed to start")
	}
}
