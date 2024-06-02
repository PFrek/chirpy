package api

import (
	"encoding/json"
	"errors"
	"github.com/PFrek/chirpy/db"
	"log"
	"net/http"
	"strconv"
)

func (config *ApiConfig) PostChirpsHandler(writer http.ResponseWriter, req *http.Request) {
	id, err := config.AuthenticateRequest(req)
	if err != nil {
		RespondWithError(writer, 401, err.Error())
		return
	}

	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		log.Printf("Error decoding parameters: %s", err)
		RespondWithError(writer, 500, "Something went wrong")
		return
	}

	if len(params.Body) > 140 {
		RespondWithError(writer, 400, "Chirp is too long")
		return
	}

	cleanedBody := replaceProfaneWords(params.Body)

	chirp, err := config.DB.CreateChirp(cleanedBody, id)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	RespondWithJSON(writer, 201, chirp)
}

func (config *ApiConfig) GetChirpsHandler(writer http.ResponseWriter, req *http.Request) {
	chirps, err := config.DB.GetChirps()
	if err != nil {
		RespondWithError(writer, 400, err.Error())
		return
	}

	RespondWithJSON(writer, 200, chirps)
}

func (config *ApiConfig) GetChirpHandler(writer http.ResponseWriter, req *http.Request) {
	id, err := strconv.Atoi(req.PathValue("id"))
	if err != nil {
		RespondWithError(writer, 400, "Invalid [id] value in path")
		return
	}

	chirp, err := config.DB.GetChirpById(id)
	if err != nil {
		if errors.Is(err, db.NotFoundError{Model: "Chirp"}) {
			RespondWithError(writer, 404, "Not Found")
			return
		}

		RespondWithError(writer, 500, err.Error())
		return
	}

	RespondWithJSON(writer, 200, chirp)
}
