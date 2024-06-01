package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/PFrek/chirpy/db"
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
		RespondWithError(writer, 500, "Something went wrong")
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
