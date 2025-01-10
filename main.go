package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
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

	apiCfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	serveMux := http.NewServeMux()
	handler := apiCfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot))))
	serveMux.Handle("/app/", handler)
	serveMux.HandleFunc("GET /api/healthz", handlerReadiness)
	serveMux.HandleFunc("GET /admin/metrics", apiCfg.handlerFileserverHits)
	serveMux.HandleFunc("POST /admin/reset", apiCfg.handlerFileserverHitsReset)
	serveMux.HandleFunc("POST /api/validate_chirp", handlerValidateChirp)

	server := http.Server{
		Addr:    ":" + port,
		Handler: serveMux,
	}
	err := server.ListenAndServe()
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

func handlerValidateChirp(w http.ResponseWriter, r *http.Request) {
	type chirp struct {
		Body string `json:"body"`
	}
	type responseSuccess struct {
		Valid bool `json:"valid"`
	}
	type responseFailure struct {
		Error string `json:"error"`
	}
	handleError := func(w http.ResponseWriter, err error, description string, statusCode int) {
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

	// Attempt to validate input json
	decoder := json.NewDecoder(r.Body)
	test := chirp{}
	err := decoder.Decode(&test)

	if err != nil {
		handleError(w, err, "Error decoding chirp", http.StatusInternalServerError)
		return
	}
	if len(test.Body) > 140 {
		handleError(w, nil, "Chirp is too long", http.StatusBadRequest)
	}

	data, err := json.Marshal(responseSuccess{
		Valid: true,
	})
	if err != nil {
		handleError(w, err, "Error marshalling response", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
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
