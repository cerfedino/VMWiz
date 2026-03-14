package confirmation

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// Package responsible of generating and verifying confirmation strings for the frontend
// e.g "Type XYZ to confirm"

func Init() {
}

// Middleware for handling confirmation tokens.
// We are interested only in updating/destructive operations (i.e not GET).
// - If ?preview=true, we respond directly with the confirmation token (the handler is never called).
// - Otherwise, we retrieve the token from the body and verify it. We return an error if it's invalid.
// - If the token is valid, we call next (the action is confirmed).
func ConfirmMiddleware(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}

		// If ?preview=true, respond with the token directly. The handler is not called.
		if r.URL.Query().Get("preview") == "true" {
			type response struct {
				ConfirmationToken string `json:"confirmationToken"`
			}
			respJSON, err := json.Marshal(response{ConfirmationToken: token})
			if err != nil {
				log.Printf("Error marshalling confirmation token: %v", err)
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(respJSON)
			return
		}

		// Otherwise we try to retrieve the confirmation token from the body and verify it
		type bodyS struct {
			ConfirmationToken string `json:"confirmationToken"`
		}
		var body bodyS
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		r.Body.Close()

		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		err = json.Unmarshal(bodyBytes, &body)
		if err != nil {
			log.Printf("Error unmarshalling request body: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		if body.ConfirmationToken != token {
			http.Error(w, "Confirmation token is invalid", 409)
			return
		}

		// Token is valid, so we proceed (the action is confirmed)
		next.ServeHTTP(w, r)
	})
}
