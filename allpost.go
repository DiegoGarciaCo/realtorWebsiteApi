package main

import (
	"net/http"
)

func (cfg *apiCfg) allPosts(w http.ResponseWriter, req *http.Request) {
	posts, err := cfg.DB.ListAllPosts(req.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusOK, posts)
}
