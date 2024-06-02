package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

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
		Password string `json:"password"`
		Email    string `json:"email"`
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
	tokenStr, err := config.generateJWTToken(user.Id)
	if err != nil {
		RespondWithError(writer, 500, "Failed to create JWT string token")
		return
	}

	// Refresh
	refresh, err := config.generateRefreshToken(user.Id)
	if err != nil {
		RespondWithError(writer, 500, fmt.Sprintf("Failed to generate refresh token: %v\n", err))
		return
	}

	response := struct {
		Id           int    `json:"id"`
		Email        string `json:"email"`
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}{
		Id:           user.Id,
		Email:        user.Email,
		Token:        tokenStr,
		RefreshToken: refresh,
	}

	RespondWithJSON(writer, 200, response)
}

func (config *ApiConfig) PostRefreshHandler(writer http.ResponseWriter, req *http.Request) {
	refresh, err := ExtractAuthorization(req)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	id, err := config.DB.ValidateRefreshToken(refresh)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	jwtToken, err := config.generateJWTToken(id)
	if err != nil {
		RespondWithError(writer, 500, "Failed to create JWT string token")
		return
	}

	response := struct {
		Token string `json:"token"`
	}{
		Token: jwtToken,
	}

	RespondWithJSON(writer, 200, response)
}

func (config *ApiConfig) PostRevokeHandler(writer http.ResponseWriter, req *http.Request) {
	refresh, err := ExtractAuthorization(req)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	_, err = config.DB.ValidateRefreshToken(refresh)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

	err = config.DB.RevokeRefreshToken(refresh)
	if err != nil {
		RespondWithError(writer, 500, fmt.Sprintf("Failed to revoke refresh token: %s\n", err.Error()))
		return
	}

	writer.WriteHeader(204)
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
	tokenStr, err := ExtractAuthorization(req)
	if err != nil {
		RespondWithError(writer, 401, "Unauthorized")
		return
	}

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
