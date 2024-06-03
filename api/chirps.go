package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/PFrek/chirpy/db"
)

type extractor[T comparable] func(string) (T, error)

func extractFilter[T comparable](name string, req *http.Request, ex extractor[T]) *T {
	var filter *T
	val, err := ex(req.URL.Query().Get(name))
	if err == nil {
		filter = new(T)
		*filter = val
	}

	return filter
}

func createChirpFilters(req *http.Request) db.ChirpFilter {
	return db.ChirpFilter{
		AuthorId: extractFilter("author_id", req, strconv.Atoi),
		Contains: extractFilter("contains", req, func(s string) (string, error) { return s, nil }),
	}
}

func (config *ApiConfig) PostChirpsHandler(writer http.ResponseWriter, req *http.Request) {
	id, err := config.AuthenticateRequest(req)
	if err != nil {
		RespondWithError(writer, 401, err.Error())
		return
	}

	type parameters struct {
		Body string `json:"body"`
	}

	params := parameters{}
	err = ExtractBody(&params, req)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
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
	chirpFilters := createChirpFilters(req)

	chirps, err := config.DB.GetChirps(chirpFilters)
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

func (config *ApiConfig) DeleteChirpHandler(writer http.ResponseWriter, req *http.Request) {
	userId, err := config.AuthenticateRequest(req)
	if err != nil {
		RespondWithError(writer, 401, err.Error())
		return
	}

	chirpId, err := strconv.Atoi(req.PathValue("id"))
	if err != nil {
		RespondWithError(writer, 400, "Invalid [id] value in path")
		return
	}

	chirp, err := config.DB.GetChirpById(chirpId)
	if err != nil {
		if errors.Is(err, db.NotFoundError{Model: "Chirp"}) {
			RespondWithError(writer, 404, "Not Found")
			return
		}

		RespondWithError(writer, 500, err.Error())
		return
	}

	if chirp.AuthorId != userId {
		RespondWithError(writer, 403, "Forbidden")
		return
	}

	err = config.DB.DeleteChirp(chirpId)
	if err != nil {
		RespondWithError(writer, 500, fmt.Sprintf("Failed to delete chirp: %s", err.Error()))
		return
	}

	writer.WriteHeader(204)
}
