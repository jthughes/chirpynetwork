package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/auth"
	"github.com/jthughes/chirpynetwork/internal/database"
)

func (cfg *apiConfig) handlerWebhookPolkaUpgraded(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Event string `json:"event"`
		Data  struct {
			UserId uuid.UUID `json:"user_id"`
		} `json:"data"`
	}

	// Authenticate request
	apiKey, err := auth.GetAPIKey(r.Header)
	if err != nil || apiKey != cfg.polkaKey {
		ResponseError(w, err, "Request not authenticated", http.StatusUnauthorized)
		return
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	req := request{}
	err = decoder.Decode(&req)
	if err != nil {
		ResponseError(w, err, "Error decoding request", http.StatusInternalServerError)
		return
	}

	if req.Event != "user.upgraded" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	_, err = cfg.db.SetUserSubscription(context.Background(), database.SetUserSubscriptionParams{
		ID:          req.Data.UserId,
		IsChirpyRed: true,
	})
	if err != nil {
		ResponseError(w, err, "User not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
