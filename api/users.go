package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PFrek/chirpy/db"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserResponse struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

func (config *ApiConfig) PostLoginHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds *int   `json:"expires_in_seconds"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		RespondWithError(writer, 500, "Something went wrong")
		return
	}

	user, err := config.DB.GetUserByEmail(params.Email)
	if err != nil {
		if errors.Is(err, db.NotFoundError{Model: "User"}) {
			RespondWithError(writer, 401, "Invalid email or password")
			return
		}

		RespondWithError(writer, 500, err.Error())
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(params.Password))
	if err != nil {
		RespondWithError(writer, 401, "Invalid email or password")
		return
	}

	// JWT
	if params.ExpiresInSeconds == nil {
		log.Println("expires_in_seconds not found, set to default 86400 (24hrs)")
		params.ExpiresInSeconds = new(int)
		*params.ExpiresInSeconds = 24 * 60 * 60 // 24 hrs in sec
	}

	if *params.ExpiresInSeconds > 24*60*60 {
		log.Println("expires_in_seconds too large, set to default 86400 (24hrs)")
		*params.ExpiresInSeconds = 24 * 60 * 60
	}

	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Duration(*params.ExpiresInSeconds) * time.Second)),
		Subject:   fmt.Sprintf("%d", user.Id),
	}

	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		claims,
	)

	tokenStr, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		RespondWithError(writer, 500, "Failed to create JWT string token")
		return
	}

	response := struct {
		Id    int    `json:"id"`
		Email string `json:"email"`
		Token string `json:"token"`
	}{
		Id:    user.Id,
		Email: user.Email,
		Token: tokenStr,
	}

	RespondWithJSON(writer, 200, response)
}

func (config *ApiConfig) PostUsersHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		RespondWithError(writer, 400, "Invalid request body")
		return
	}

	if len(params.Email) == 0 || len(params.Password) == 0 {
		RespondWithError(writer, 400, "Email and password are required")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(params.Password), 4)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	user, err := config.DB.CreateUser(params.Email, string(hashed))
	if err != nil {
		if errors.Is(err, db.ExistingEmailError{}) {
			RespondWithError(writer, 400, err.Error())
			return
		}
		RespondWithError(writer, 500, err.Error())
		return
	}

	responseUser := UserResponse{
		Id:    user.Id,
		Email: user.Email,
	}

	RespondWithJSON(writer, 201, responseUser)
}

func (config *ApiConfig) PutUsersHandler(writer http.ResponseWriter, req *http.Request) {
	tokenSplit := strings.Split(req.Header.Get("Authorization"), " ")

	if len(tokenSplit) != 2 {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	tokenStr := tokenSplit[1]

	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.JWTSecret), nil
	})
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	idStr, err := token.Claims.GetSubject()
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
	}

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		RespondWithError(writer, 400, "Invalid request body")
		return
	}

	if len(params.Email) == 0 || len(params.Password) == 0 {
		RespondWithError(writer, 400, "Email and password are required")
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(params.Password), 4)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	user, err := config.DB.UpdateUser(db.User{
		Id:       id,
		Email:    params.Email,
		Password: string(hashed),
	})

	if err != nil {
		if errors.Is(err, db.ExistingEmailError{}) {
			RespondWithError(writer, 400, err.Error())
			return
		}
		RespondWithError(writer, 500, err.Error())
		return
	}

	responseUser := UserResponse{
		Id:    user.Id,
		Email: user.Email,
	}

	RespondWithJSON(writer, 200, responseUser)
}

func (config *ApiConfig) GetUsersHandler(writer http.ResponseWriter, req *http.Request) {
	users, err := config.DB.GetUsers()
	if err != nil {
		RespondWithError(writer, 400, err.Error())
		return
	}

	responseUsers := []UserResponse{}
	for _, user := range users {
		responseUsers = append(responseUsers, UserResponse{
			Id:    user.Id,
			Email: user.Email,
		})
	}

	RespondWithJSON(writer, 200, responseUsers)
}

func (config *ApiConfig) GetUserHandler(writer http.ResponseWriter, req *http.Request) {
	id, err := strconv.Atoi(req.PathValue("id"))
	if err != nil {
		RespondWithError(writer, 400, "Invalid [id] value in path")
		return
	}

	user, err := config.DB.GetUserById(id)
	if err != nil {
		if errors.Is(err, db.NotFoundError{Model: "User"}) {
			RespondWithError(writer, 404, "Not Found")
			return
		}

		RespondWithError(writer, 500, err.Error())
		return
	}

	responseUser := UserResponse{
		Id:    user.Id,
		Email: user.Email,
	}

	RespondWithJSON(writer, 200, responseUser)
}
