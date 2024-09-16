package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting Authenticate handler...")
	log.Println("Incoming request: ", r.Method, r.URL.Path)
	log.Println("Forwarding request to authentication-service...")
	app.ForwardRequest(w, r, "http://authentication-service/authenticate", http.MethodPost)
}

func (app *Config) CreateSnippet(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting CreateSnippet handler...")
	// Log the incoming request details
	log.Println("Incoming request: ", r.Method, r.URL.Path)
	log.Println("Forwarding request to snippet-service...")
	app.ForwardRequest(w, r, "http://snippet-service/snippet", http.MethodPost)
}

func (app *Config) SearchSnippet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	reqURL := "http://snippet-service/search?query=" + url.QueryEscape(query)

	log.Println("Forwarding request to snippet-service search...")
	app.ForwardRequest(w, r, reqURL, http.MethodGet)
}

func (app *Config) LogItem(w http.ResponseWriter, r *http.Request) {
	// Forward to the log service
	url := "http://logger-service/log"
	app.ForwardRequest(w, r, http.MethodPost, url)
}

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	// Forward to the mail service
	url := "http://comm-service/send"
	app.ForwardRequest(w, r, http.MethodPost, url)
}

// ForwardRequest forwards the request to the appropriate microservice
func (app *Config) ForwardRequest(w http.ResponseWriter, r *http.Request, serviceURL string, method string) {
	// Create a new request with the same body as the original request
	var bodyBytes []byte
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}
	defer r.Body.Close()

	req, err := http.NewRequest(method, serviceURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// Forward headers from the original request to the microservice
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		app.errorJSON(w, err)
		return
	}
	defer resp.Body.Close()

	// Forward the response from the microservice back to the client
	var jsonResponse jsonResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResponse)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	if jsonResponse.Error {
		app.errorJSON(w, errors.New(jsonResponse.Message), resp.StatusCode)
		return
	}

	app.writeJSON(w, resp.StatusCode, jsonResponse)
}
