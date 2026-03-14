package main

import (
	"log"
	"net/http"
)

func main() {
	port := "8080"
	mux := http.NewServeMux()
	server := http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// StripPrefix removes "/app" from the path so FileServer sees "/" or "/index.html" etc.
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		msg := "OK"
		w.Write([]byte(msg))
	})
	// mux.Handle("assets/logo.png", http.FileServer(http.Dir("./assets/logo.png")))
	log.Printf("Serving on port: %s\n", port)
	server.ListenAndServe()

}
