package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/auth"
	"github.com/jthughes/chirpynetwork/internal/database"
)

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding login request", http.StatusInternalServerError)
		return
	}

	// Validate login
	dbUser, err := cfg.db.GetUserByEmail(context.Background(), test.Email)
	if err == nil {
		err = auth.CheckPasswordHash(test.Password, dbUser.HashedPassword)
	}
	if err != nil {
		ResponseError(w, nil, "Incorrect email or password", http.StatusUnauthorized)
		return
	}

	// Create access token
	accessToken, err := auth.MakeAccessToken(dbUser.ID, cfg.secretKey)
	if err != nil {
		ResponseError(w, err, "Error creating authentication token", http.StatusInternalServerError)
		return
	}

	// Create refresh token
	refreshExpiry := 60 * 24 * time.Hour
	if err != nil {
		ResponseError(w, err, "Error setting refresh expiry", http.StatusInternalServerError)
		return
	}
	refreshTokenString, err := auth.MakeRefreshToken()
	if err != nil {
		ResponseError(w, err, "Error creating refresh token", http.StatusInternalServerError)
		return
	}

	refreshToken, err := cfg.db.StoreRefreshToken(context.Background(), database.StoreRefreshTokenParams{
		Token:     refreshTokenString,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().Add(refreshExpiry),
	})
	if err != nil {
		ResponseError(w, err, "Error storing refresh token", http.StatusInternalServerError)
		return
	}

	// Build response
	type response struct {
		ID           uuid.UUID `json:"id"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
		Email        string    `json:"email"`
		AccessToken  string    `json:"token"`
		RefreshToken string    `json:"refresh_token"`
	}

	data, err := json.Marshal(response{
		ID:           dbUser.ID,
		CreatedAt:    dbUser.CreatedAt,
		UpdatedAt:    dbUser.UpdatedAt,
		Email:        dbUser.Email,
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		ResponseError(w, err, "Authorization token not found", http.StatusUnauthorized)
		return
	}
	dbToken, err := cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		ResponseError(w, err, "Unable to retrieve refresh token from db", http.StatusUnauthorized)
		return
	}
	if dbToken.ExpiresAt.Before(time.Now()) || dbToken.RevokedAt.Valid {
		ResponseError(w, err, "Refresh token expired", http.StatusUnauthorized)
		return
	}

	accessToken, err := auth.MakeAccessToken(dbToken.UserID, cfg.secretKey)
	if err != nil {
		ResponseError(w, err, "Error creating authentication token", http.StatusInternalServerError)
		return
	}

	type response struct {
		AccessToken string `json:"token"`
	}
	data, err := json.Marshal(response{
		AccessToken: accessToken,
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		ResponseError(w, err, "Authorization token not found", http.StatusNotFound)
		return
	}
	dbToken, err := cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		ResponseError(w, err, "Unable to retrieve refresh token from db", http.StatusNotFound)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(context.Background(), database.RevokeRefreshTokenParams{
		Token: dbToken.Token,
		RevokedAt: sql.NullTime{
			Time:  time.Now(),
			Valid: true,
		},
	})
	if err != nil {
		ResponseError(w, err, "Unable to revoke refresh token", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}