package main

import (
	"fmt"
	"log"
	"net/http"
)

type apiConfig struct {
	fileserverHits int
}

func (config *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		config.fileserverHits++
		fmt.Printf("Incremented fileserverHits to %v\n", config.fileserverHits)
		next.ServeHTTP(writer, req)
	})
}

func (config *apiConfig) metricsHandler(writer http.ResponseWriter, req *http.Request) {
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	fmt.Fprintf(writer, "Hits: %v", config.fileserverHits)
}

func (config *apiConfig) resetHandler(writer http.ResponseWriter, req *http.Request) {
	config.fileserverHits = 0
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("Fileserver hits counter reset to 0"))
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	var apiConfig apiConfig

	mux := http.NewServeMux()
	fileserverHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/*", apiConfig.middlewareMetricsInc(fileserverHandler))

	mux.HandleFunc("/healthz", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(200)
		writer.Write([]byte("OK"))
	})

	mux.HandleFunc("/metrics", apiConfig.metricsHandler)

	mux.HandleFunc("/reset", apiConfig.resetHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
