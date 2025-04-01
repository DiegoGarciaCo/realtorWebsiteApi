package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiCfg) UpdateThumbnail(w http.ResponseWriter, req *http.Request) {
	UUID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
		return
	}

	err = req.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "File too big", http.StatusBadRequest)
		return
	}

	files := req.MultipartForm.File["file"]
	if len(files) == 0 || len(files) > 1 {
		http.Error(w, "Exactly one image file is required", http.StatusBadRequest)
		return
	}
	fileHeader := files[0]

	file, err := fileHeader.Open()
	if err != nil {
		http.Error(w, "Could not open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")

	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "File is not an image", http.StatusBadRequest)
		return
	}

	tx, err := cfg.SQLDB.BeginTx(req.Context(), nil)
	if err != nil {
		http.Error(w, "Could not begin transaction", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := cfg.DB.WithTx(tx)

	post, err := qtx.GetPostThumbnailById(req.Context(), UUID)
	if err != nil {
		http.Error(w, "Post not found", http.StatusNotFound)
		return
	}

	key := getAssetPath(contentType, "thumbnails")

	_, err = cfg.S3Client.PutObject(req.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.S3Bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})

	if err != nil { 
		http.Error(w, "Failed to upload image to S3", http.StatusInternalServerError)
		return
	}
	imageURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.S3Bucket, cfg.S3Region, key)

	err = qtx.UpdatePostThumbnail(req.Context(), database.UpdatePostThumbnailParams{
		ID:       UUID,
		Thumbnail: sql.NullString{String: imageURL, Valid: imageURL != ""},
	})

	if err != nil {
		deleteFromS3(cfg, req.Context(), imageURL)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		deleteFromS3(cfg, req.Context(), imageURL)
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	err = deleteFromS3(cfg, req.Context(), post.Thumbnail.String)

	if err != nil {
		http.Error(w, "Could not delete old image", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}