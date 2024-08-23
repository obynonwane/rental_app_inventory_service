package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type jsonResponse struct {
	Error      bool   `json:"error"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
	Data       any    `json:"data,omitempty"`
}

// read json
func (app *Config) readJSON(w http.ResponseWriter, r *http.Request, data any) error {
	//add a limiation on the uploaded json file
	maxByte := 104876
	//validate to make sure the request body is not more than 1 byte
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxByte))
	//decode the request body
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(data)
	if err != nil {
		return err
	}

	//check that there is only a single json value in the data we received
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must have only a single json value")
	}

	return nil
}

// write json
func (app *Config) writeJSON(w http.ResponseWriter, status int, data any, headers ...http.Header) error {

	//converts the passed data into json representative
	out, err := json.Marshal(data)

	if err != nil {
		return err
	}

	//check if any header is supplied and set the respnse header
	if len(headers) > 0 {
		for key, value := range headers[0] {
			w.Header()[key] = value
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err = w.Write(out)
	if err != nil {
		return err
	}

	return nil
}

//generate error json response

func (app *Config) errorJSON(w http.ResponseWriter, err error, data any, status ...int) error {
	statusCode := http.StatusBadRequest
	if len(status) > 0 {
		statusCode = status[0]
	}

	var payload jsonResponse
	payload.Error = true
	payload.Message = err.Error()
	payload.StatusCode = statusCode
	payload.Data = data

	return app.writeJSON(w, statusCode, payload)
}
