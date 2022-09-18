package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit broker",
	}

	_ = app.writeJson(w, http.StatusOK, payload)
}

func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	switch requestPayload.Action {
	case "auth":
		app.auth(w, requestPayload.Auth)
	case "log":
		app.LogItem(w, requestPayload.Log)
	default:
		app.errorJson(w, errors.New("Unknown action"))
	}
}

func (app *Config) LogItem(w http.ResponseWriter, entry LogPayload) {
	jsonData, _ := json.MarshalIndent(entry, "", "\t")

	logServiceUrl := "http://logger-service/log"

	request, err := http.NewRequest(http.MethodPost, logServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		app.errorJson(w, err)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	app.writeJson(w, http.StatusAccepted, payload)
}

func (app *Config) auth(w http.ResponseWriter, a AuthPayload) {
	jsonData, _ := json.MarshalIndent(a, "", "\t")

	request, err := http.NewRequest("POST", "http://auth-service/auth", bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, err)
		return
	}

	client := &http.Client{}
	res, err := client.Do(request)
	if err != nil {
		app.errorJson(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusUnauthorized {
		app.errorJson(w, errors.New("invalid credentials"))
		return
	} else if res.StatusCode != http.StatusAccepted {
		app.errorJson(w, errors.New("error calling service"))
		return
	}

	var jsonFromService jsonResponse
	err = json.NewDecoder(res.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJson(w, err)
		return
	}

	if jsonFromService.Error {
		app.errorJson(w, err, http.StatusUnauthorized)
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "authed"
	payload.Data = jsonFromService.Data

	app.writeJson(w, http.StatusAccepted, payload)
}
