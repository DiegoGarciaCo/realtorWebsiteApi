package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

func (cfg *apiCfg) uploadThumnail(w http.ResponseWriter, req *http.Request) {
	err := req.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "File too big", http.StatusBadRequest)
		return
	}

	UUID, err := uuid.Parse(req.PathValue("id"))
	if err != nil {
		http.Error(w, "Invalid UUID", http.StatusBadRequest)
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
		if err = file.Close(); err != nil {
			log.Fatal("Failed to close file", err)
			return
		}
		http.Error(w, "Only image files are allowed", http.StatusBadRequest)
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

	err = cfg.DB.UpdatePostThumbnail(req.Context(), database.UpdatePostThumbnailParams{
		ID:        UUID,
		Thumbnail: sql.NullString{String: imageURL, Valid: true},
	})

	if err != nil {
		cleanupS3(cfg, req.Context(), []string{key})
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}
