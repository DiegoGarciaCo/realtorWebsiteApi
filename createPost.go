package main

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
)

func (cfg *apiCfg) createDraftPost(w http.ResponseWriter, req *http.Request) {
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
