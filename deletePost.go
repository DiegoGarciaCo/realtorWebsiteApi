package main

import (
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiCfg) deletePost(w http.ResponseWriter, req *http.Request) {
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
