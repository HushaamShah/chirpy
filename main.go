package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	queries        *database.Queries
	platform       string
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserId    uuid.UUID `json:"user_id"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")

	db, err := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)
	port := "8080"
	mux := http.NewServeMux()
	cfg := apiConfig{
		fileserverHits: atomic.Int32{},
		queries:        dbQueries,
		platform:       platform,
	}
	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /api/healthz", handlerHealth)
	mux.HandleFunc("GET /admin/metrics", cfg.getMetrics)
	mux.HandleFunc("GET /api/chirps", cfg.getChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", cfg.getSingleChirp)
	mux.HandleFunc("POST /admin/reset", cfg.resetMetrics)
	mux.HandleFunc("POST /api/users", cfg.addUser)
	mux.HandleFunc("POST /api/chirps", cfg.addChirp)
	log.Printf("Serving on port: %s\n", port)
	server.ListenAndServe()

}

func handlerHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	msg := "OK"
	w.Write([]byte(msg))
}

func validateChirp(chirp string) (string, error) {

	if len(chirp) > 140 {
		// respondWithError(w, 400, "Chirp is too long")
		return "", fmt.Errorf("Chirp is too long")
	}

	chirp = replaceProfanity(chirp)

	return chirp, nil

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
	test := msg
	profanity := [3]string{"kerfuffle", "sharbert", "fornax"}
	msgArray := strings.Split(msg, " ")
	for i, word := range msgArray {
		for _, prof := range profanity {
			if strings.EqualFold(word, prof) {
				msgArray[i] = strings.ReplaceAll(strings.ToLower(word), strings.ToLower(prof), "****")
			}
		}
	}
	msg = strings.Join(msgArray, " ")
	if msg == test {
		fmt.Println("same")
	}
	return msg
	// strings.Replace(msg, )
}

func (cfg *apiConfig) addUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	params := parameters{}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, err.Error())
		fmt.Println(err)
	}
	fmt.Println("params ADD USER")

	fmt.Println(params)
	user, err := cfg.queries.CreateUser(r.Context(), params.Email)
	respUser := User{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email:     user.Email,
	}
	fmt.Println("respUser")
	fmt.Println(respUser)
	respondWithJSON(w, 201, respUser)

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
	if strings.ToLower(cfg.platform) != "dev" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
	}
	cfg.fileserverHits.Store(0)
	err := cfg.queries.DeleteAllUsers(r.Context())
	// err = cfg.queries.DeleteAllChirps(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
}

func (cfg *apiConfig) addChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}

	params := parameters{}
	fmt.Println("r.Body")
	fmt.Println(r.Body)
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, err.Error())
		fmt.Println(err)
	}
	fmt.Println("params add chirps user id")
	fmt.Println(params.UserId)
	userUUID, err := uuid.Parse(params.UserId)
	if err != nil {
		respondWithError(w, 500, err.Error())
		fmt.Println(err)
	}
	fmt.Println("user ID")
	fmt.Println(userUUID)

	params.Body, err = validateChirp(params.Body)

	args := database.CreateChirpParams{
		Body:   params.Body,
		UserID: userUUID,
	}

	chirp, err := cfg.queries.CreateChirp(r.Context(), args)
	respChirp := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserId:    chirp.UserID,
	}
	respondWithJSON(w, 201, respChirp)
}

func (cfg *apiConfig) getChirps(w http.ResponseWriter, r *http.Request) {
	chirps, err := cfg.queries.GetChirps(r.Context())
	if err != nil {
		fmt.Println("Error at Getting Chirps")
		respondWithError(w, 500, "Error at Getting Chirps")
		return
	}
	chirpsResp := []Chirp{}
	for _, v := range chirps {
		chirp := Chirp{
			Body:      v.Body,
			ID:        v.ID,
			CreatedAt: v.CreatedAt,
			UpdatedAt: v.UpdatedAt,
			UserId:    v.UserID,
		}
		chirpsResp = append(chirpsResp, chirp)
	}
	respondWithJSON(w, 200, chirpsResp)
}

func (cfg *apiConfig) getSingleChirp(w http.ResponseWriter, r *http.Request) {
	chirpId := r.PathValue("chirpID")
	chirpUUID, err := uuid.Parse(chirpId)
	if err != nil {
		respondWithError(w, 500, err.Error())
		fmt.Println(err)
	}
	chirp, err := cfg.queries.GetSingleChirp(r.Context(), chirpUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			fmt.Println(err)
			respondWithError(w, 404, "No chirp found!")
			return
		}
		fmt.Println(err)
		respondWithError(w, 500, "Error at Getting Chirp")
		return
	}

	chirpResp := Chirp{
		Body:      chirp.Body,
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		UserId:    chirp.UserID,
	}
	respondWithJSON(w, 200, chirpResp)
}
