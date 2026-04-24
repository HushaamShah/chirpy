package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
)

func (cfg *apiConfig) addChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body   string `json:"body"`
		UserId string `json:"user_id"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	userUUID, err := uuid.Parse(params.UserId)
	if err != nil {
		respondWithError(w, 500, err.Error())
		return
	}
	validatedBody, err := validateChirp(params.Body)
	if err != nil {
		respondWithError(w, 400, err.Error())
		return
	}
	args := database.CreateChirpParams{
		Body:   validatedBody,
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
		return
	}
	chirp, err := cfg.queries.GetSingleChirp(r.Context(), chirpUUID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			respondWithError(w, 404, "No chirp found!")
			return
		}
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
