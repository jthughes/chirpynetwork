package main

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/database"
)

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

func dbChirpToChirp(dbChirp database.Chirp) Chirp {
	return Chirp{
		ID:        dbChirp.ID,
		CreatedAt: dbChirp.CreatedAt,
		UpdatedAt: dbChirp.UpdatedAt,
		Body:      dbChirp.Body,
		UserId:    dbChirp.UserID,
	}
}

func clean_message(message string) string {
	profane := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(message, " ")
	cleaned := []string{}
	for _, word := range words {
		if slices.Contains(profane, strings.ToLower(word)) {
			cleaned = append(cleaned, "****")
		} else {
			cleaned = append(cleaned, word)
		}
	}
	return strings.Join(cleaned, " ")
}

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body string `json:"body"`
	}
	type responseSuccess struct {
		CleanedBody string `json:"cleaned_body"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	if len(test.Body) > 140 {
		ResponseError(w, nil, "Chirp is too long", http.StatusBadRequest)
		return
	}

	data, err := json.Marshal(responseSuccess{
		CleanedBody: clean_message(test.Body),
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerNewChirp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body   string    `json:"body"`
		UserId uuid.UUID `json:"user_id"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.GetUserById(context.Background(), test.UserId)
	if err != nil {
		ResponseError(w, nil, "User does not exist", http.StatusBadRequest)
		return
	}

	if len(test.Body) > 140 {
		ResponseError(w, nil, "Chirp is too long", http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.db.CreateChirp(context.Background(), database.CreateChirpParams{
		ID:        uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Body:      test.Body,
		UserID:    user.ID,
	})
	if err != nil {
		ResponseError(w, nil, "Unable to create chirp", http.StatusInternalServerError)
		return
	}

	chirp := dbChirpToChirp(dbChirp)
	data, err := json.Marshal(chirp)
	SetJSONResponse(w, http.StatusCreated, data, err)
}

func (cfg *apiConfig) handlerGetAllChirps(w http.ResponseWriter, r *http.Request) {

	chirps, err := cfg.db.GetAllChirps(context.Background())
	if err != nil {
		ResponseError(w, err, "Error retrieving chirps", http.StatusInternalServerError)
		return
	}

	out := []Chirp{}
	for _, item := range chirps {
		out = append(out, dbChirpToChirp(item))
	}

	data, err := json.Marshal(out)
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {

	requestUserID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		ResponseError(w, err, "Error parsing chirp id", http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.db.GetChirpByID(context.Background(), requestUserID)
	if err != nil {
		ResponseError(w, err, "Chirp ID not found", http.StatusNotFound)
		return
	}

	chirp := dbChirpToChirp(dbChirp)
	data, err := json.Marshal(chirp)
	SetJSONResponse(w, http.StatusOK, data, err)
}
