package handlers

import (
	"database/sql"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/keighl/postmark"
)

type apiCfg struct {
	Port             string
	Secret           string
	DB               *database.Queries
	SQLDB            *sql.DB
	S3Client         *s3.Client
	S3Bucket         string
	S3Region         string
	Env              string
	BetterAuthSecret string
	postmarkClient   *postmark.Client
	EmailSecret      []byte
	FromEmail        string
	CRMAPIKey        string
}

func NewConfig(port, secret, s3Bucket, s3Region, env string, db *database.Queries, sqlDB *sql.DB, s3Client *s3.Client, betterAuthSecret string, postmarkClient *postmark.Client, emailSecret []byte, fromEmail string, crmAPIKey string) *apiCfg {
	return &apiCfg{
		Port:             port,
		Secret:           secret,
		DB:               db,
		SQLDB:            sqlDB,
		S3Client:         s3Client,
		S3Bucket:         s3Bucket,
		S3Region:         s3Region,
		Env:              env,
		BetterAuthSecret: betterAuthSecret,
		postmarkClient:   postmarkClient,
		EmailSecret:      emailSecret,
		FromEmail:        fromEmail,
		CRMAPIKey:        crmAPIKey,
	}
}
