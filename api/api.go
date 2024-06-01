package api

import (
	"encoding/json"
	"fmt"
	"github.com/PFrek/chirpy/db"
	"log"
	"net/http"
	"slices"
	"strings"
)

type ApiConfig struct {
	fileserverHits int
	DB             *db.DB
	JWTSecret      string
}

func (config *ApiConfig) MiddlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, req *http.Request) {
		config.fileserverHits++
		fmt.Printf("Incremented fileserverHits to %v\n", config.fileserverHits)
		next.ServeHTTP(writer, req)
	})
}

func (config *ApiConfig) MetricsHandler(writer http.ResponseWriter, req *http.Request) {
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

func (config *ApiConfig) ResetHandler(writer http.ResponseWriter, req *http.Request) {
	config.fileserverHits = 0
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.WriteHeader(200)
	writer.Write([]byte("Fileserver hits counter reset to 0"))
}

func RespondWithError(writer http.ResponseWriter, code int, msg string) {
	type errBody struct {
		Error string `json:"error"`
	}
	respBody := errBody{
		Error: msg,
	}

	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		writer.WriteHeader(500)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(dat)
}

func RespondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		writer.WriteHeader(500)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(dat)

}

func replaceProfaneWords(chirp string) string {
	badWords := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	replacedWords := []string{}
	for _, word := range strings.Split(chirp, " ") {
		if slices.Contains(badWords, strings.ToLower(word)) {
			replacedWords = append(replacedWords, strings.Repeat("*", 4))
			continue
		}

		replacedWords = append(replacedWords, word)
	}

	return strings.Join(replacedWords, " ")
}
