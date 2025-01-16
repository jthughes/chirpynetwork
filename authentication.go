package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/auth"
	"github.com/jthughes/chirpynetwork/internal/database"
)

func (cfg *apiConfig) authenticateRequest(r *http.Request) (uuid.UUID, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("access token not found")
	}
	userId, err := auth.ValidateJWT(token, cfg.secretKey)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid access token")
	}
	return userId, nil
}

func (cfg *apiConfig) authenticateRefresh(r *http.Request) (string, uuid.UUID, error) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		return "", uuid.UUID{}, fmt.Errorf("invalid authorization header")
	}
	dbToken, err := cfg.db.GetRefreshToken(r.Context(), token)
	if err != nil {
		return "", uuid.UUID{}, fmt.Errorf("refresh token not found")
	}
	if dbToken.ExpiresAt.Before(time.Now()) || dbToken.RevokedAt.Valid {
		return "", uuid.UUID{}, fmt.Errorf("refresh token expired")
	}
	return dbToken.Token, dbToken.UserID, nil
}

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
		User
		AccessToken  string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	data, err := json.Marshal(response{
		User:         dbUserToUser(dbUser),
		AccessToken:  accessToken,
		RefreshToken: refreshToken.Token,
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {

	_, userID, err := cfg.authenticateRefresh(r)
	if err != nil {
		ResponseError(w, err, "Invalid refresh token", http.StatusUnauthorized)
		return
	}

	accessToken, err := auth.MakeAccessToken(userID, cfg.secretKey)
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
	token, _, err := cfg.authenticateRefresh(r)
	if err != nil {
		ResponseError(w, err, "Invalid refresh token", http.StatusNotFound)
		return
	}

	_, err = cfg.db.RevokeRefreshToken(context.Background(), database.RevokeRefreshTokenParams{
		Token: token,
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
