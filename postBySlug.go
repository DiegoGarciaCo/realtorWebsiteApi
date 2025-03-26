package main

import (
	"net/http"
)

func (cfg *apiCfg) postBySlug(w http.ResponseWriter, req *http.Request) {
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
