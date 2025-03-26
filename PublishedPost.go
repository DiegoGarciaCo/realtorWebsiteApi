package main

import (
	"net/http"
)

func (cfg *apiCfg) PublishedPost(w http.ResponseWriter, req *http.Request) {
	post, err := cfg.DB.ListPublishedPosts(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}
	respondWithJSON(w, http.StatusOK, post)
}
