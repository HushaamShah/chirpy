package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func main() {
	port := "8080"
	mux := http.NewServeMux()
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
	}
	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", handlerHealth)
	mux.HandleFunc("GET /admin/metrics", cfg.getMetrics)
	mux.HandleFunc("POST /admin/reset", cfg.resetMetrics)
	mux.HandleFunc("POST /api/validate_chirp", validateChirp)
	log.Printf("Serving on port: %s\n", port)
	server.ListenAndServe()

}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	msg := "OK"
	w.Write([]byte(msg))
}

func validateChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, 400, "Chirp is too long")
		return
	}

	type resSuccVal struct {
		Cleaned_body string `json:"string"`
	}

	res := resSuccVal{
		Cleaned_body: params.Body,
	}

	res.Cleaned_body = replaceProfanity(res.Cleaned_body)
	// println(res)
	respondWithJSON(w, 200, res)

}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type resErrorVal struct {
		Error string `json:"error"`
	}

	resError := resErrorVal{}
	log.Printf("Error decoding parameters: %s", msg)
	resError.Error = msg
	dat, err := json.Marshal(resError)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	// resError := resErrorVal{}
	// log.Printf("Error decoding parameters: %s", msg)
	// resError.error = msg
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(dat)
}

func replaceProfanity(msg string) string {
	profanity := [3]string{"kerfuffle", "sharbert", "fornax"}
	for _, prof := range profanity {
		msg = strings.ReplaceAll(strings.ToLower(msg), strings.ToLower(prof), "****")
	}
	return msg
	// strings.Replace(msg, )
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	hits := cfg.fileserverHits.Load()
	str := fmt.Sprintf("<html> <body> <h1>Welcome, Chirpy Admin</h1> <p>Chirpy has been visited %d times!</p> </body> </html>", hits)
	w.Write([]byte(str))
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}
