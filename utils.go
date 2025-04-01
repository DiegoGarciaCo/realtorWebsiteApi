package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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
