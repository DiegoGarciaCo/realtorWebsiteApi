package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiCfg) PublishPost(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	if id != "" {
		UUID, err := uuid.Parse(id)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid UUID", err)
			return
		}
		err = cfg.DB.UpdatePostStatus(req.Context(), database.UpdatePostStatusParams{
			ID:     UUID,
			Status: sql.NullString{String: "published", Valid: true},
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
