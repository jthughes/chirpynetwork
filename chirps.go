package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
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

func (cfg *apiConfig) handlerNewChirp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Body string `json:"body"`
	}

	userId, err := cfg.authenticateRequest(r)
	if err != nil {
		ResponseError(w, err, "Invalid token", http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	test := request{}
	err = decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	user, err := cfg.db.GetUserById(context.Background(), userId)
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
	author := r.URL.Query().Get("author_id")

	var chirps []database.Chirp
	var err error
	if author == "" {
		chirps, err = cfg.db.GetAllChirps(context.Background())
		if err != nil {
			ResponseError(w, err, "Error retrieving chirps", http.StatusNotFound)
			return
		}
	} else {
		authorID, err := uuid.Parse(author)
		if err != nil {
			ResponseError(w, err, "Error parsing author id", http.StatusBadRequest)
		}
		chirps, err = cfg.db.GetAllChirpsFromAuthor(context.Background(), authorID)
		if err != nil {
			ResponseError(w, err, "Error retrieving chirps", http.StatusNotFound)
			return
		}
	}

	sortDir := r.URL.Query().Get("sort")
	if sortDir == "desc" {
		sort.Slice(chirps, func(i, j int) bool { return chirps[i].CreatedAt.After(chirps[j].CreatedAt) })
	}

	out := []Chirp{}
	for _, item := range chirps {
		out = append(out, dbChirpToChirp(item))
	}

	data, err := json.Marshal(out)
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerGetChirpByID(w http.ResponseWriter, r *http.Request) {

	requestChirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		ResponseError(w, err, "Error parsing chirp id", http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.db.GetChirpByID(context.Background(), requestChirpID)
	if err != nil {
		ResponseError(w, err, "Chirp ID not found", http.StatusNotFound)
		return
	}

	chirp := dbChirpToChirp(dbChirp)
	data, err := json.Marshal(chirp)
	SetJSONResponse(w, http.StatusOK, data, err)
}

func (cfg *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {

	userID, err := cfg.authenticateRequest(r)
	if err != nil {
		ResponseError(w, err, "User not authenticated", http.StatusUnauthorized)
		return
	}

	chirpID, err := uuid.Parse(r.PathValue("chirpID"))
	if err != nil {
		ResponseError(w, err, "Error parsing chirp id", http.StatusBadRequest)
		return
	}

	dbChirp, err := cfg.db.GetChirpByID(context.Background(), chirpID)
	if err != nil {
		ResponseError(w, err, "Chirp ID not found", http.StatusNotFound)
		return
	}

	if dbChirp.UserID != userID {
		ResponseError(w, err, "User not authorized", http.StatusForbidden)
		return
	}

	err = cfg.db.DeleteChirp(context.Background(), chirpID)
	if err != nil {
		ResponseError(w, err, "Error deleting chirp", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
