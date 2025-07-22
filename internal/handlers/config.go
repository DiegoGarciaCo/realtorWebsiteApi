package handlers

import (
	"database/sql"
	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type apiCfg struct {
	Port        string
	Secret      string
	DB          *database.Queries
	SQLDB       *sql.DB
	AppPassword string
	FUBKey      string
	System      string
	SystemKey   string
	S3Client    *s3.Client
	S3Bucket    string
	S3Region    string
	Env         string
}

func NewConfig(port, secret, appPassword, fubkey, system, systemKey, s3Bucket, s3Region, env string, db *database.Queries, sqlDB *sql.DB, s3Client *s3.Client) *apiCfg {
	return &apiCfg{
		Port:        port,
		Secret:      secret,
		DB:          db,
		SQLDB:       sqlDB,
		AppPassword: appPassword,
		FUBKey:      fubkey,
		System:      system,
		SystemKey:   systemKey,
		S3Client:    s3Client,
		S3Bucket:    s3Bucket,
		S3Region:    s3Region,
		Env:         env,
	}
}
