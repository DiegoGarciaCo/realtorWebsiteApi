package handlers

import (
	"database/sql"
	"net/url"
	"encoding/json"
	"fmt"
	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"log"
	"net/http"
	"strings"
	"time"
)

func (cfg *apiCfg) PublishedPost(w http.ResponseWriter, req *http.Request) {
	post, err := cfg.DB.ListPublishedPosts(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	respondWithJSON(w, http.StatusOK, post)
}

func (cfg *apiCfg) AllPosts(w http.ResponseWriter, req *http.Request) {
	type resParams struct {
		ID      string   `json:"ID"`
		Title   string   `json:"Title"`
		Slug    string   `json:"Slug"`
		Excerpt sql.NullString`json:"Excerpt"`
		Author  sql.NullString `json:"Author"`
		PublishedAt sql.NullTime`json:"PublishedAt"`
		Status sql.NullString `json:"Status"`
		Thumbnail sql.NullString `json:"Thumbnail"`
		CreatedAt sql.NullTime `json:"CreatedAt"`
		Tags    []string `json:"Tags"`
	}
	posts, err := cfg.DB.ListAllPosts(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	res := make([]resParams, len(posts))
	for i, post := range posts {
		res[i] = resParams{
			ID:          post.ID.String(),
			Title:       post.Title,
			Slug:        post.Slug,
			Excerpt:     post.Excerpt,
			Author:      post.Author,
			PublishedAt: post.PublishedAt,
			Status:      post.Status,
			Thumbnail:   post.Thumbnail,
			CreatedAt:   post.CreatedAt,
			Tags:        post.Tags,
		}
	}

	respondWithJSON(w, http.StatusOK, res)
}

func (cfg *apiCfg) CreateDraftPost(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Title   string   `json:"title"`
		Slug    string   `json:"slug"`
		Content string   `json:"content"`
		Excerpt string   `json:"excerpt"`
		Author  string   `json:"author"`
		Tags    []string `json:"tags"`
	}

	decoder := json.NewDecoder(req.Body)
	params := reqParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	post, err := cfg.DB.CreatePost(req.Context(), database.CreatePostParams{
		Title:   params.Title,
		Slug:    params.Slug,
		Content: params.Content,
		Excerpt: sql.NullString{String: params.Excerpt, Valid: params.Excerpt != ""},
		Author:  sql.NullString{String: params.Author, Valid: params.Author != ""},
		Status:  sql.NullString{String: "draft", Valid: true},
		Tags:    params.Tags,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, post)
}

func (cfg *apiCfg) DeletePost(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")

	UUID, err := uuid.Parse(id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid UUID", err)
		return
	}

	post, err := cfg.DB.DeletePost(req.Context(), UUID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if !post.Thumbnail.Valid {
		respondWithJSON(w, http.StatusNoContent, nil)
		return
	}

	err = deleteFromS3(cfg, req.Context(), post.Thumbnail.String)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not delete image from S3", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) PostBySlug(w http.ResponseWriter, req *http.Request) {
	slug := req.PathValue("slug")
	if slug == "" {
		respondWithError(w, http.StatusBadRequest, "Missing slug", nil)
		return
	}
	post, err := cfg.DB.GetPostBySlug(req.Context(), slug)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Post not found", err)
		return
	}
	respondWithJSON(w, http.StatusOK, post)
}

func (cfg *apiCfg) PublishPost(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id != "" {
		UUID, err := uuid.Parse(id)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid UUID", err)
			return
		}
		err = cfg.DB.PublishPost(req.Context(), database.PublishPostParams{
			ID: UUID,
			Status: sql.NullString{
				String: "published",
				Valid:  true,
			},
		})

		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
			return
		}
		respondWithJSON(w, http.StatusOK, "Post published")
	}

	type reqParams struct {
		Title   string   `json:"title"`
		Slug    string   `json:"slug"`
		Content string   `json:"content"`
		Excerpt string   `json:"excerpt"`
		Author  string   `json:"author"`
		Tags    []string `json:"tags"`
	}

	decoder := json.NewDecoder(req.Body)
	params := reqParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	post, err := cfg.DB.CreatePost(req.Context(), database.CreatePostParams{
		Title:       params.Title,
		Slug:        params.Slug,
		Content:     params.Content,
		Excerpt:     sql.NullString{String: params.Excerpt, Valid: params.Excerpt != ""},
		Author:      sql.NullString{String: params.Author, Valid: params.Author != ""},
		PublishedAt: sql.NullTime{Time: time.Now(), Valid: true},
		Status:      sql.NullString{String: "published", Valid: true},
		Tags:        params.Tags,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusOK, post)
}

func (cfg *apiCfg) SaveAndPublishPost(w http.ResponseWriter, req *http.Request) {
	type resParams struct {
		Title   string   `json:"title"`
		Slug    string   `json:"slug"`
		Content string   `json:"content"`
		Excerpt string   `json:"excerpt"`
		Author  string   `json:"author"`
		Tags    []string `json:"tags"`
	}

	id := req.PathValue("id")
	UUID, err := uuid.Parse(id)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid UUID", err)
		return
	}

	decoder := json.NewDecoder(req.Body)
	params := resParams{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	_, err = cfg.DB.SaveAndPublishPost(req.Context(), database.SaveAndPublishPostParams{
		ID:      UUID,
		Title:   params.Title,
		Slug:    params.Slug,
		Content: params.Content,
		Excerpt: sql.NullString{String: params.Excerpt, Valid: params.Excerpt != ""},
		Author:  sql.NullString{String: params.Author, Valid: params.Author != ""},
		Status:  sql.NullString{String: "published", Valid: true},
		Tags:    params.Tags,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) UpdatePost(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		ID      string   `json:"id"`
		Title   string   `json:"title"`
		Slug    string   `json:"slug"`
		Content string   `json:"content"`
		Author  string   `json:"author"`
		Excerpt string   `json:"excerpt"`
		Status  string   `json:"status"`
		Tags    []string `json:"tags"`
	}

	decoder := json.NewDecoder(req.Body)
	params := reqParams{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	UUID, err := uuid.Parse(params.ID)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid UUID", err)
		return
	}

	post, err := cfg.DB.UpdatePost(req.Context(), database.UpdatePostParams{
		ID:      UUID,
		Title:   params.Title,
		Slug:    params.Slug,
		Content: params.Content,
		Author:  sql.NullString{String: params.Author, Valid: params.Author != ""},
		Excerpt: sql.NullString{String: params.Excerpt, Valid: params.Excerpt != ""},
		Status:  sql.NullString{String: params.Status, Valid: params.Status != ""},
		Tags:    params.Tags,
	})

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusOK, post)
}

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
		ID:        UUID,
		Thumbnail: sql.NullString{String: imageURL, Valid: imageURL != ""},
	})

	if err != nil {
		err = deleteFromS3(cfg, req.Context(), imageURL)
		if err != nil {
			log.Printf("Failed to delete image from S3: %v", err)
		}
		http.Error(w, "Something went wrong", http.StatusInternalServerError)
		return
	}

	err = tx.Commit()
	if err != nil {
		err = deleteFromS3(cfg, req.Context(), imageURL)
		if err != nil {
			log.Printf("Failed to delete image from S3 after commit failure: %v", err)
		}
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

func (cfg *apiCfg) UploadThumnail(w http.ResponseWriter, req *http.Request) {
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

func (cfg *apiCfg) GetPostsByCategory(w http.ResponseWriter, req *http.Request) {
	type resParams struct {
		ID          string         `json:"ID"`
		Title       string         `json:"Title"`
		Slug        string         `json:"Slug"`
		Excerpt     sql.NullString `json:"Excerpt"`
		Author      sql.NullString `json:"Author"`
		PublishedAt sql.NullTime   `json:"PublishedAt"`
		Thumbnail   sql.NullString `json:"Thumbnail"`
		CreatedAt   sql.NullTime   `json:"CreatedAt"`
		Tags        []string       `json:"Tags"`
	}

	encodedCategory := req.PathValue("category")
	if encodedCategory == "" {
		respondWithError(w, http.StatusBadRequest, "Missing category", nil)
		return
	}

	category := url.QueryEscape(encodedCategory)

	posts, err := cfg.DB.GetPostByCategory(req.Context(), []string{category})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	if len(posts) == 0 {
		respondWithJSON(w, http.StatusNotFound, "No posts found for this category")
		return
	}

	res := make([]resParams, len(posts))
	for i, post := range posts {
		res[i] = resParams{
			ID:          post.ID.String(),
			Title:       post.Title,
			Slug:        post.Slug,
			Excerpt:     post.Excerpt,
			Author:      post.Author,
			PublishedAt: post.PublishedAt,
			Thumbnail:   post.Thumbnail,
			CreatedAt:   post.CreatedAt,
			Tags:        post.Tags,
		}
	}

	respondWithJSON(w, http.StatusOK, res)
}
