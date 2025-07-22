package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/DiegoGarciaCo/websitesAPI/internal/auth"
	"github.com/DiegoGarciaCo/websitesAPI/internal/database"
)

func (cfg *apiCfg) Login(w http.ResponseWriter, req *http.Request) {
	type reqParams struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	credentials := reqParams{}
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&credentials)
	if err != nil {
		log.Printf("Error decoding request: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not decode request", err)
		return
	}

	user, err := cfg.DB.GetUserByUsername(req.Context(), credentials.Username)
	if err != nil {
		log.Printf("Error getting user: %s", err)
		respondWithError(w, http.StatusUnauthorized, "Could not find user with that username", err)
		return
	}

	err = auth.CheckPasswordHash(credentials.Password, user.PasswordHash.String)
	if err != nil {
		log.Printf("Error checking password: %s", err)
		respondWithError(w, http.StatusUnauthorized, "Incorrect username or password", err)
		return
	}

	token, err := auth.MakeJWT(user.ID, cfg.Secret, time.Hour)
	if err != nil {
		log.Printf("Error making token: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not generate token", err)
		return
	}

	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		log.Printf("Error making refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not generate refresh token", err)
		return
	}

	csrfToken, err := auth.MakeToken()
	if err != nil {
		log.Printf("Error making csrf token: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not generate csrf token", err)
		return
	}

	tx, err := cfg.SQLDB.BeginTx(req.Context(), nil)
	if err != nil {
		log.Printf("Error beginning transaction: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not begin transaction", err)
		return
	}
	defer tx.Rollback()

	qtx := cfg.DB.WithTx(tx)

	err = qtx.StorecsfToken(req.Context(), database.StorecsfTokenParams{
		Token:  csrfToken,
		UserID: user.ID,
	})
	if err != nil {
		log.Printf("Error storing csrf token: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not store csrf token", err)
		return
	}

	err = qtx.StoreRefreshToken(req.Context(), database.StoreRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		log.Printf("Error storing refresh token: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not store refresh token", err)
		return
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %s", err)
		respondWithError(w, http.StatusInternalServerError, "Could not commit transaction", err)
		return
	}
	domain := "localhost"
	var secure bool
	if cfg.Env != "dev" {
		domain = "soldbyghost.com"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    token,
		Expires:  time.Now().Add(time.Hour),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    refreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Value:    csrfToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: false,
		Secure:   secure,
	})
	log.Print("User logged in")

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) Logout(w http.ResponseWriter, req *http.Request) {
	refreshToken, err := req.Cookie("refreshToken")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "not logged in", err)
		return
	}
	csrf, err := req.Cookie("csrfToken")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "no same site token", err)
		return
	}

	tx, err := cfg.SQLDB.BeginTx(req.Context(), nil)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not begin transaction", err)
		return
	}
	defer tx.Rollback()

	qtx := cfg.DB.WithTx(tx)

	err = qtx.RevokeRefreshToken(req.Context(), refreshToken.Value)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not revoke refresh token", err)
		return
	}

	err = qtx.DeletecsfToken(req.Context(), csrf.Value)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not delete csrf token", err)
		return
	}
	domain := "localhost"
	var secure bool
	if cfg.Env == "dev" {
		domain = "soldbbyghost.com"
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: false,
		Secure:   secure,
	})

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) RefreshToken(w http.ResponseWriter, req *http.Request) {
	domain := "localhost"
	var secure bool
	if cfg.Env == "dev" {
		domain = "soldbbyghost.com"
		secure = false
	}
	refreshToken, err := req.Cookie("refreshToken")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusBadRequest, "Could not find refresh token", err)
		return
	}

	token, err := cfg.DB.GetRefreshToken(req.Context(), refreshToken.Value)
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusBadRequest, "Could not find refresh token", err)
		return
	}

	if time.Now().After(token.ExpiresAt) {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusUnauthorized, "Refresh token has expired", nil)
		return
	}

	if token.RevokedAt.Valid {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusUnauthorized, "Refresh token has been revoked", nil)
		return
	}

	user, err := cfg.DB.GetUserByID(req.Context(), token.UserID)
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusInternalServerError, "Could not find user", err)
		return
	}

	JwtToken, err := auth.MakeJWT(user.ID, cfg.Secret, time.Hour)
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusInternalServerError, "Could not generate JWT token", err)
		return
	}

	newRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusInternalServerError, "Could not generate refresh token", err)
		return
	}

	csrfToken, err := auth.MakeToken()
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "refreshToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "token",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: true,
			Secure:   secure,
		})
		http.SetCookie(w, &http.Cookie{
			Name:     "csrfToken",
			Value:    "",
			MaxAge:   -1,
			Path:     "/",
			SameSite: http.SameSiteLaxMode,
			Domain:   domain,
			HttpOnly: false,
			Secure:   secure,
		})
		respondWithError(w, http.StatusInternalServerError, "Could not generate csrf token", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    JwtToken,
		Expires:  time.Now().Add(time.Hour),
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		Domain:   domain,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    newRefreshToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Domain:   domain,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
		Secure:   secure,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "csrfToken",
		Value:    csrfToken,
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Domain:   domain,
		Path:     "/",
		SameSite: http.SameSiteLaxMode,
		HttpOnly: false,
		Secure:   secure,
	})

	respondWithJSON(w, http.StatusNoContent, nil)
}

func (cfg *apiCfg) ValidateJWT(w http.ResponseWriter, req *http.Request) {
	token, err := req.Cookie("token")
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "not logged in", err)
		return
	}

	userID, err := auth.ValidateJWT(token.Value, cfg.Secret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token", err)
		return
	}

	_, err = cfg.DB.GetUserByID(req.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "invalid token", err)
		return
	}

	respondWithJSON(w, http.StatusNoContent, nil)
}
