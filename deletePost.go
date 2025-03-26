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

	err = cfg.DB.DeletePost(req.Context(), UUID)

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong", err)
		return
	}

	respondWithJSON(w, http.StatusOK, "Post deleted")
}
