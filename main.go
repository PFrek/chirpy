package main

import (
	"log"
	"net/http"

	"github.com/PFrek/chirpy/api"
	"github.com/PFrek/chirpy/db"
)

func main() {
	const filepathRoot = "."
	const port = "8080"
	const dbPath = "database.json"

	var apiConfig api.ApiConfig
	db, err := db.NewDB(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	apiConfig.DB = db

	mux := http.NewServeMux()
	fileserverHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/*", apiConfig.MiddlewareMetricsInc(fileserverHandler))

	mux.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(200)
		writer.Write([]byte("OK\n"))
	})

	mux.HandleFunc("POST /api/chirps", apiConfig.PostChirpsHandler)
	mux.HandleFunc("GET /api/chirps", apiConfig.GetChirpsHandler)
	mux.HandleFunc("GET /api/chirps/{id}", apiConfig.GetChirpHandler)

	mux.HandleFunc("/api/reset", apiConfig.ResetHandler)

	mux.HandleFunc("GET /admin/metrics", apiConfig.MetricsHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
