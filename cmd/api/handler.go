package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayload `json:"mail,omitempty"`
}

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayload struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {
	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting CreateSnippet handler...")

	// Log the incoming request details
	log.Println("Incoming request: ", r.Method, r.URL.Path)

	// Set the headers
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		log.Println("Missing Content-Type header")
		app.errorJSON(w, errors.New("content-type header missing"), http.StatusBadRequest)
		return
	}
	log.Println("Original Content-Type:", contentType)

	// Create the request to the snippet service
	log.Println("Creating request for snippet-service...")
	request, err := http.NewRequest(http.MethodPost, "http://snippet-service/snippet", r.Body)
	if err != nil {
		log.Println("Error creating request:", err)
		app.errorJSON(w, err)
		return
	}

	request.Header.Set("Accept", "*")
	request.Header.Set("Content-Type", contentType)

	// Log the request to snippet service
	client := &http.Client{}
	log.Println("Sending request to snippet-service...")
	response, err := client.Do(request)
	if err != nil {
		log.Println("Error sending request to snippet-service:", err)
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// Log the response from the snippet service
	log.Printf("Response received: Status=%d, StatusText=%s", response.StatusCode, response.Status)

	if response.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(response.Body)
		log.Printf("error response from snippet-service: %s", string(bodyBytes))
		app.errorJSON(w, errors.New("error response from snippet-service"), response.StatusCode)
		return
	}

	// Create a variable, read response.Body into it
	log.Println("Decoding response from snippet-service...")
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		log.Println("Error decoding response:", err)
		app.errorJSON(w, err)
		return
	}

	// Check if snippet-service returned an error
	if jsonFromService.Error {
		log.Println("Snippet-service returned an error:", jsonFromService.Message)
		app.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// Log successful response
	log.Println("Snippet saved successfully:", jsonFromService.Message)

	// Send the success response to the client
	var payload jsonResponse
	payload.Error = false
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	log.Println("Sending success response to client...")
	app.writeJSON(w, http.StatusOK, payload)
	log.Println("CreateSnippet handler finished successfully.")
}

func (app *Config) SearchSnippet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	// Construct the URL with query params
	reqURL := fmt.Sprintf("http://snippet-service/search?query=%s", url.QueryEscape(query))

	// Call the snippet microservice
	request, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer response.Body.Close()

	// Handle the response
	var jsonFromService jsonResponse
	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Check for an error response from the microservice
	if jsonFromService.Error {
		app.errorJSON(w, err, http.StatusInternalServerError)
		return
	}

	// Send back the response
	var payload jsonResponse
	payload.Error = false
	payload.Message = jsonFromService.Message
	payload.Data = jsonFromService.Data

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) LogItem(w http.ResponseWriter, r *http.Request) {
	var requestPayload LogPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	jsonData, err := json.MarshalIndent(requestPayload, "", "\t")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	logServiceUrl := "http://logger-service/log"
	req, err := http.NewRequest("POST", logServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted {
		app.errorJSON(w, errors.New("failed to log item"))
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"

	app.writeJSON(w, http.StatusOK, payload)
}

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	var requestPayload MailPayload

	err := app.readJSON(w, r, &requestPayload)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	jsonData, err := json.MarshalIndent(requestPayload, "", "\t")
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	commServiceUrl := "http://comm-service/send"

	req, err := http.NewRequest("POST", commServiceUrl, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		app.errorJSON(w, errors.New("error calling mail service"))
		return
	}

	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + requestPayload.To

	app.writeJSON(w, http.StatusOK, payload)
}
