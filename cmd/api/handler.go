package main

import (
	"bytes"
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

func (app *Config) GetUserSnippets(w http.ResponseWriter, r *http.Request) {
	reqURL := "http://snippet-service/user/snippets"

	log.Println("forwarding request to snippet-service get user snippets...")
	app.ForwardRequest(w, r, reqURL, http.MethodGet)
}

func (app *Config) LogItem(w http.ResponseWriter, r *http.Request) {
	// Forward to the log service
	var log LogPayload
	err := app.readJSON(w, r, &log)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	// app.logEventViaRabbit(w, log)
}

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	// Forward to the mail service
	url := "http://comm-service/send"
	app.ForwardRequest(w, r, url, http.MethodPost)
}

// ForwardRequest forwards the request to the appropriate microservice
func (app *Config) ForwardRequest(w http.ResponseWriter, r *http.Request, serviceURL string, method string) {
	// Log the forwarded service URL
	log.Printf("Forwarding request to %s with method %s", serviceURL, method)

	// Read the body into a buffer
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("error reading request body:", err)
		app.errorJSON(w, err)
		return
	}
	defer r.Body.Close() // Always close the request body

	// Log request body for debugging
	log.Printf("Request body: %s", string(bodyBytes))

	// Create a new request with the same body as the original request
	req, err := http.NewRequest(method, serviceURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Println("error creating new request:", err)
		app.errorJSON(w, err)
		return
	}

	// Forward headers from the original request to the microservice
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	// Log headers for debugging
	log.Println("Forwarding headers:")
	for key, values := range req.Header {
		log.Printf("%s: %s", key, values)
	}

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("error forwarding request to microservice:", err)
		app.errorJSON(w, err)
		return
	}
	defer resp.Body.Close()

	// Log the response status
	log.Printf("Response status: %d", resp.StatusCode)

	// Forward the response from the microservice back to the client
	w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	w.WriteHeader(resp.StatusCode)
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Println("error forwarding response body:", err)
	}
}

// func (app *Config) logEventViaRabbit(w http.ResponseWriter, l LogPayload) {
// 	err := app.pushToQueue(l.Name, l.Data)
// 	if err != nil {
// 		app.errorJSON(w, err)
// 		return
// 	}

// 	var payload jsonResponse
// 	payload.Error = false
// 	payload.Message = "logged via RabbitMQ"

// 	app.writeJSON(w, http.StatusOK, payload)
// }

// func (app *Config) pushToQueue(name, msg string) error {
// 	emitter, err := event.NewEventEmitter(app.Rabbit)
// 	if err != nil {
// 		return err
// 	}

// 	payload := LogPayload{
// 		Name: name,
// 		Data: msg,
// 	}

// 	j, _ := json.Marshal(&payload)

// 	err = emitter.Push(string(j), "log.INFO")
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
