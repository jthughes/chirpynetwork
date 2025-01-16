package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/joho/godotenv"
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
	serveMux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	serveMux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)

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
