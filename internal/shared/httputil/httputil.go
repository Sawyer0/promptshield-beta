package httputil

import (
	"encoding/json"
	"io"
	"net/http"
)

// WriteJSON writes a JSON response
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response
func WriteError(w http.ResponseWriter, statusCode int, message string, err error) {
	response := map[string]interface{}{
		"error":  message,
		"status": statusCode,
	}

	if err != nil {
		response["details"] = err.Error()
	}

	WriteJSON(w, statusCode, response)
}

// ReadBody reads the request body
func ReadBody(r *http.Request) ([]byte, error) {
	return io.ReadAll(r.Body)
}
