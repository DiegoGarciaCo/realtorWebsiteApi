package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiCfg) SaveAndPublishPost(w http.ResponseWriter, req *http.Request) {
	type resParams struct {
		Title   string `json:"title"`
		Slug    string `json:"slug"`
		Content string `json:"content"`
		Excerpt string `json:"excerpt"`
		Author  string `json:"author"`
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