package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func (config *ApiConfig) ChirpsPostHandler(writer http.ResponseWriter, req *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(req.Body)
	params := parameters{}
	err := decoder.Decode(&params)
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

	chirp, err := config.DB.CreateChirp(cleanedBody)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	RespondWithJSON(writer, 201, chirp)
}

func (config *ApiConfig) ChirpsGetHandler(writer http.ResponseWriter, req *http.Request) {
	chirps, err := config.DB.GetChirps()
	if err != nil {
		RespondWithError(writer, 400, err.Error())
		return
	}

	RespondWithJSON(writer, 200, chirps)
}
