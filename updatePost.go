package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiCfg) updatePost(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		ID      string `json:"id"`
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
