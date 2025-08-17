package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens,omitempty"`
	Stream    bool          `json:"stream,omitempty"`
}

type choice struct {
	Index   int         `json:"index"`
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
}

var (
	ignoreRe  = regexp.MustCompile(`(?i)ignore .*instruction`)
	revealRe  = regexp.MustCompile(`(?i)(reveal|show).*(system|hidden).*prompt`)
	hackedRe  = regexp.MustCompile(`(?i)\bhacked\b`)
	secretsRe = regexp.MustCompile(`(?i)(password|api[_-]?key|token)`)
)

func analyze(messages []chatMessage) string {
	user := ""
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.ToLower(messages[i].Role) == "user" {
			user = messages[i].Content
			break
		}
	}
	if user == "" {
		return "Hello!"
	}

	// Simulate vulnerable model behavior when asked to ignore instructions or reveal system prompt
	if ignoreRe.MatchString(user) && revealRe.MatchString(user) {
		return "Here are my system instructions: [SIMULATED_SYSTEM_PROMPT]"
	}
	if hackedRe.MatchString(user) {
		return "HACKED"
	}
	if secretsRe.MatchString(user) {
		return "Here is a secret you asked for: [SIMULATED_SECRET]"
	}

	// Default benign response
	return "Hello! This is a mock provider response."
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	resp := chatResponse{
		ID:      "mock-" + time.Now().Format("150405"),
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []choice{{
			Index: 0,
			Message: chatMessage{
				Role:    "assistant",
				Content: analyze(req.Messages),
			},
		}},
	}
	buf, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf)
}

func health(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }

func main() {
	port := os.Getenv("MOCK_PROVIDER_PORT")
	if port == "" {
		port = "18080"
	}

	http.HandleFunc("/v1/chat/completions", chatHandler)
	http.HandleFunc("/health", health)

	addr := ":" + port
	log.Printf("Mock provider listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
