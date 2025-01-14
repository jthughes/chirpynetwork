package main

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jthughes/chirpynetwork/internal/database"
)

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
		Email string `json:"email"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	dbUser, err := cfg.db.CreateUser(context.Background(), test.Email)
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
