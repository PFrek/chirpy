package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/PFrek/chirpy/db"
	"github.com/golang-jwt/jwt/v5"
)

type ApiConfig struct {
	fileserverHits int
	DB             *db.DB
	JWTSecret      string
	PolkaKey       string
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

func ExtractBody(params interface{}, req *http.Request) error {
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		return fmt.Errorf("Something went wrong: %s", err.Error())
	}

	return nil
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

func (config *ApiConfig) generateRefreshToken(id int) (string, error) {
	data := make([]byte, 32)
	_, err := rand.Read(data)
	if err != nil {
		return "", err
	}

	hex := hex.EncodeToString(data)

	err = config.DB.CreateRefreshToken(hex, id)
	if err != nil {
		return "", err
	}

	return hex, nil
}

func (config *ApiConfig) generateJWTToken(id int) (string, error) {
	expiration := time.Duration(1) * time.Hour

	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiration)),
		Subject:   fmt.Sprint(id),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenStr, err := token.SignedString([]byte(config.JWTSecret))
	return tokenStr, err

}

func ExtractAuthorization(req *http.Request) (string, error) {
	tokenSplit := strings.Split(req.Header.Get("Authorization"), " ")

	if len(tokenSplit) != 2 {
		return "", errors.New("Unauthorized")
	}

	tokenStr := tokenSplit[1]
	return tokenStr, nil
}

func (config *ApiConfig) AuthenticateRequest(req *http.Request) (int, error) {
	tokenStr, err := ExtractAuthorization(req)
	if err != nil {
		return 0, errors.New("Unauthorized")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})
	if err != nil {
		return 0, errors.New("Unauthorized")
	}

	idStr, err := token.Claims.GetSubject()
	if err != nil {
		return 0, errors.New("Unauthorized")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, errors.New("Unauthorized")
	}

	return id, nil
}

func (config *ApiConfig) AuthenticatePolkaKey(req *http.Request) error {
	key, err := ExtractAuthorization(req)
	if err != nil {
		return errors.New("Unauthorized")
	}

	if key != config.PolkaKey {
		return errors.New("Unauthorized")
	}

	return nil
}
