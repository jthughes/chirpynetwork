package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/auth"
	"github.com/jthughes/chirpynetwork/internal/database"
)

// handlerLogin expects this to not have a copy of Password
type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

func dbUserToUser(dbUser database.User) User {
	return User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}
}

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}
	hashed_password, err := auth.HashPassword(test.Password)
	if err != nil {
		ResponseError(w, err, "Error hashing password", http.StatusInternalServerError)
		return
	}
	dbUser, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          test.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		ResponseError(w, err, "Error creating user", http.StatusInternalServerError)
		return
	}
	user := dbUserToUser(dbUser)
	data, err := json.Marshal(user)
	SetJSONResponse(w, http.StatusCreated, data, err)
}

func (cfg *apiConfig) handlerUpdateLogin(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}
	hashed_password, err := auth.HashPassword(test.Password)
	if err != nil {
		ResponseError(w, err, "Error hashing password", http.StatusInternalServerError)
		return
	}
	dbUser, err := cfg.db.CreateUser(context.Background(), database.CreateUserParams{
		Email:          test.Email,
		HashedPassword: hashed_password,
	})
	if err != nil {
		ResponseError(w, err, "Error creating user", http.StatusInternalServerError)
		return
	}
	user := dbUserToUser(dbUser)
	data, err := json.Marshal(user)
	SetJSONResponse(w, http.StatusCreated, data, err)
}

func (cfg *apiConfig) handlerResetUsers(w http.ResponseWriter, r *http.Request) {

	type responseSuccess struct {
		Deleted bool `json:"deleted"`
	}

	if cfg.platform != "dev" {
		ResponseError(w, nil, "Cannot reset users outside of development", http.StatusForbidden)
		return
	}

	err := cfg.db.DeleteAllUsers(context.Background())
	if err != nil {
		ResponseError(w, err, "Error deleting all users", http.StatusInternalServerError)
	}

	data, err := json.Marshal(responseSuccess{
		Deleted: true,
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}
