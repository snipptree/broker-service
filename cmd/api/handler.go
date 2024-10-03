package main

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
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

func (app *Config) GetSnippetById(w http.ResponseWriter, r *http.Request) {
	log.Println("Starting GetSnippetById handler...")
	// Log the incoming request details
	log.Println("Incoming request: ", r.Method, r.URL.Path)
	log.Println("Forwarding request to snippet-service...")

	snippetID := chi.URLParam(r, "id")
	log.Println(snippetID)
	app.ForwardRequest(w, r, "http://snippet-service/snippet/"+snippetID, http.MethodGet)
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

func (app *Config) LikeSnippet(w http.ResponseWriter, r *http.Request) {
	snippetID := chi.URLParam(r, "id")
	reqURL := "http://snippet-service/snippet/" + snippetID + "/like"

	log.Println("forwarding request to snippet-service like snippet...")
	app.ForwardRequest(w, r, reqURL, http.MethodPost)
}

func (app *Config) UnlikeSnippet(w http.ResponseWriter, r *http.Request) {
	snippetID := chi.URLParam(r, "id")
	reqURL := "http://snippet-service/snippet/" + snippetID + "/unlike"

	log.Println("forwarding request to snippet-service unlike snippet...")
	app.ForwardRequest(w, r, reqURL, http.MethodDelete)
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
	log.Printf("Forwarding request to %s with method %s", serviceURL, method)

	// Only read the body for methods that require it
	var bodyBytes []byte
	var err error
	if method != http.MethodGet {
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			log.Println("error reading request body:", err)
			app.errorJSON(w, err)
			return
		}
		defer r.Body.Close()
	}

	req, err := http.NewRequest(method, serviceURL, bytes.NewReader(bodyBytes))
	if err != nil {
		log.Println("error creating new request:", err)
		app.errorJSON(w, err)
		return
	}
	req.Close = true

	// Forward headers from the original request to the microservice
	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{
		Timeout: 30 * time.Second, // Set a longer timeout if needed
	}

	log.Println("req")
	log.Println(req)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error forwarding request: %v", err)
		app.errorJSON(w, err)
		return
	}
	defer resp.Body.Close()

	// Log the response status
	log.Printf("Received response status: %d", resp.StatusCode)

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
