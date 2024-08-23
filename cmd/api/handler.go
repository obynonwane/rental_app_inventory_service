package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
)

func (app *Config) GetUsers(w http.ResponseWriter, r *http.Request) {

	users, err := app.Repo.GetAll()
	if err != nil {
		if err == sql.ErrNoRows {
			app.errorJSON(w, errors.New("no record found"), nil, http.StatusBadRequest)
			return
		}

		app.errorJSON(w, err, nil, http.StatusInternalServerError)
		return
	}

	log.Println(users)

	payload := jsonResponse{
		Error:      false,
		StatusCode: http.StatusAccepted,
		Message:    "users retrieved successfully",
		Data:       users,
	}

	app.writeJSON(w, http.StatusAccepted, payload)
}
