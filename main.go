package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/jthughes/chirpynetwork/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	db             *database.Queries
	platform       string
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
	}
	serveMux := http.NewServeMux()
	handler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	serveMux.Handle("/app/", handler)
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerFileserverHits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerResetUsers)
	serveMux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)
	serveMux.HandleFunc("POST /api/users", apiCfg.handlerCreateUser)

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

func handlerReadiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(http.StatusText(http.StatusOK)))
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

func ResponseError(w http.ResponseWriter, err error, description string, statusCode int) {
	type responseFailure struct {
		Error string `json:"error"`
	}
	// Log if server error
	if err != nil {
		log.Printf("%s: %s", description, err)
	} else {
		log.Printf("%s", description)
	}

	// Attempt to create formated error message
	data, err := json.Marshal(responseFailure{
		Error: fmt.Sprintf("%s: %s", description, err),
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

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type responseSuccess struct {
		CleanedBody string `json:"cleaned_body"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := chirp{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	if len(test.Body) > 140 {
		ResponseError(w, nil, "Chirp is too long", http.StatusBadRequest)
	}

	data, err := json.Marshal(responseSuccess{
		CleanedBody: clean_message(test.Body),
	})
	SetJSONResponse(w, http.StatusOK, data, err)
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

func (cfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email string `json:"email"`
	}
	type responseSuccess struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := request{}
	err := decoder.Decode(&test)
	if err != nil {
		ResponseError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}

	dbuser, err := cfg.db.CreateUser(context.Background(), test.Email)
	if err != nil {
		ResponseError(w, err, "Error creating user", http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(responseSuccess{
		ID:        dbuser.ID,
		CreatedAt: dbuser.CreatedAt,
		UpdatedAt: dbuser.UpdatedAt,
		Email:     dbuser.Email,
	})
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
