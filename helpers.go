package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

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

func validateChirp(chirp string) (string, error) {

	if len(chirp) > 140 {
		// respondWithError(w, 400, "Chirp is too long")
		return "", fmt.Errorf("Chirp is too long")
	}

	chirp = replaceProfanity(chirp)

	return chirp, nil

}
