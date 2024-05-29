package main

import (
	"encoding/json"
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
	writer.Header().Set("Content-Type", "text/html")
	writer.WriteHeader(200)
	template := "<html>\n"
	template += "<body>\n"
	template += "<h1>Welcome, Chirpy Admin</h1>\n"
	template += fmt.Sprintf("<p>Chirpy has been visited %d times!</p>\n", config.fileserverHits)
	template += "</body>\n"
	template += "</html>\n"
	writer.Write([]byte(template))
}

func (config *apiConfig) resetHandler(writer http.ResponseWriter, req *http.Request) {
	config.fileserverHits = 0
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("Fileserver hits counter reset to 0"))
}

func (config *apiConfig) returnErrorResponse(errorMsg string, status int, writer http.ResponseWriter) {
	type errBody struct {
		Error string `json:"error"`
	}
	respBody := errBody{
		Error: errorMsg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		writer.WriteHeader(500)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	writer.Write(dat)
}

func (config *apiConfig) chirpValidationHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		config.returnErrorResponse("Something went wrong", 500, writer)
		return
	}

	if len(params.Body) > 140 {
		config.returnErrorResponse("Chirp is too long", 400, writer)
		return
	}

	respBody := struct {
		Valid bool `json:"valid"`
	}{
		Valid: true,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		writer.WriteHeader(500)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(200)
	writer.Write(dat)
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	var apiConfig apiConfig

	mux := http.NewServeMux()
	fileserverHandler := http.StripPrefix("/app", http.FileServer(http.Dir(filepathRoot)))
	mux.Handle("/app/*", apiConfig.middlewareMetricsInc(fileserverHandler))

	mux.HandleFunc("GET /api/healthz", func(writer http.ResponseWriter, req *http.Request) {
		writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
		writer.WriteHeader(200)
		writer.Write([]byte("OK\n"))
	})

	mux.HandleFunc("POST /api/validate_chirp", apiConfig.chirpValidationHandler)

	mux.HandleFunc("/api/reset", apiConfig.resetHandler)

	mux.HandleFunc("GET /admin/metrics", apiConfig.metricsHandler)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port: %s\n", filepathRoot, port)
	log.Fatal(server.ListenAndServe())
}
