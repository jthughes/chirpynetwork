package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/jthughes/chirpynetwork/internal/auth"
	"github.com/jthughes/chirpynetwork/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
	secretKey      string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			cfg.fileserverHits.Add(1)
			next.ServeHTTP(w, r)
		})
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Unable to open connection to database: %s\n", err)
		os.Exit(1)
	}
	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
		db:             database.New(db),
		platform:       os.Getenv("PLATFORM"),
		secretKey:      os.Getenv("SECRET_KEY"),
	}
	serveMux := http.NewServeMux()
	handler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	serveMux.Handle("/app/", handler)

	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerFileserverHits)

	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerResetUsers)
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

	// serveMux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	serveMux.HandleFunc("POST /api/chirps", apiCfg.handlerNewChirp)
	serveMux.HandleFunc("GET /api/chirps", apiCfg.handlerGetAllChirps)
	serveMux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirpByID)

	serveMux.HandleFunc("POST /api/login", apiCfg.handlerLogin)

	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	err = server.ListenAndServe()
	if err != nil {
		fmt.Printf("Failed to start server: %s\n", err)
		return
	}
}

func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds string `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding login request", http.StatusInternalServerError)
		return
	}

	// Check expiry
	expiry, err := strconv.Atoi(test.ExpiresInSeconds)
	if err != nil || expiry < 1 || expiry > 60 {
		expiry = 60
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

	// Create authentication token
	expiryDuration, err := time.ParseDuration(fmt.Sprintf("%ds", expiry))
	if err != nil {
		ResponseError(w, err, "Error setting expiry", http.StatusInternalServerError)
		return
	}
	token, err := auth.MakeJWT(dbUser.ID, cfg.secretKey, expiryDuration)
	if err != nil {
		ResponseError(w, err, "Error creating authentication token", http.StatusInternalServerError)
		return
	}

	// Build response
	type response struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
		Token     string    `json:"token"`
	}

	data, err := json.Marshal(response{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Token:     token,
	})
	SetJSONResponse(w, http.StatusOK, data, err)
}

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
}

func ResponseError(w http.ResponseWriter, err error, description string, statusCode int) {
	type responseFailure struct {
		Error string `json:"error"`
	}
	// Log if server error
	var error_msg string
	if err != nil {
		error_msg = fmt.Sprintf("%s: %s", description, err)
	} else {
		error_msg = description
	}
	log.Println(error_msg)

	// Attempt to create formated error message
	data, err := json.Marshal(responseFailure{
		Error: error_msg,
	})

	// Just error out if fail to create formatted error message
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error marshalling JSON: %s", err)
		return
	}

	// Write formatted error to response
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func SetJSONResponse(w http.ResponseWriter, statusCode int, jsonData []byte, err error) {
	if err != nil {
		ResponseError(w, err, "Error marshalling response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (cfg *apiConfig) handlerFileserverHitsReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0"))
}

func (cfg *apiConfig) handlerFileserverHits(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())))
}
