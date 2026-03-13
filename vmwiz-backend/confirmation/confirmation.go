package confirmation

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// Package responsible of generating and verifying confirmation strings for the frontend
// e.g "Type XYZ to confirm"

const acceptedToken string = "yes"

const ConfirmationTokenContextField string = "confirmationToken"

func Init() {
}

/**
 * Middleware for handling confirmation tokens.
 * We are interested only in updating/destructive operations (i.e not GET).
 * - If ?preview=true, we generate a new confirmation token and put it in the request's context
 * - otherwise, we retrieve the token from the body and verify. We return an error if anything goes wrong (e.g invalid token, etc...)
 * After this middleware if a token is added to the context, then the action is NOT confirmed and the token has to be sent back to the user such that he can supply it later on.
 */
func ConfirmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			next.ServeHTTP(w, r)
			return
		}

		// If ?preview=true, we add a token to the context and proceed.
		if r.URL.Query().Get("preview") == "true" {
			var token = acceptedToken
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ConfirmationTokenContextField, token)))
			return
		}

		// Otherwise we try to retrieve the confirmation token from the body and we try to verify it
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

		if body.ConfirmationToken != acceptedToken {
			http.Error(w, "Confirmation token is invalid", 409)
			return
		}

		// Token is valid, so we proceed the request without adding anything to the context (i.e the action is confirmed)
		next.ServeHTTP(w, r)
	})
}
