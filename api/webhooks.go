package api

import (
	"errors"
	"net/http"

	"github.com/PFrek/chirpy/db"
)

func (config *ApiConfig) PolkaWebhookHandler(writer http.ResponseWriter, req *http.Request) {
	err := config.AuthenticatePolkaKey(req)
	if err != nil {
		RespondWithError(writer, 401, err.Error())
		return
	}

	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserId int `json:"user_id"`
		} `json:"data"`
	}

	params := parameters{}
	err = ExtractBody(&params, req)
	if err != nil {
		RespondWithError(writer, 500, err.Error())
		return
	}

	if params.Event != "user.upgraded" {
		writer.WriteHeader(204)
		return
	}

	_, err = config.DB.UpgradeUser(params.Data.UserId)
	if err != nil {
		if errors.Is(err, db.NotFoundError{Model: "User"}) {
			RespondWithError(writer, 404, "Not Found")
			return
		}

		RespondWithError(writer, 500, err.Error())
	}

	writer.WriteHeader(204)
}
