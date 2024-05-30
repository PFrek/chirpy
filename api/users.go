package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func (config *ApiConfig) PostUsersHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		RespondWithError(writer, 500, "Something went wrong")
		return
	}

	user, err := config.DB.CreateUser(params.Email)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	RespondWithJSON(writer, 201, user)
}
