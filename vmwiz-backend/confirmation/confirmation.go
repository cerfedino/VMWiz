package confirmation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func ConfirmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			next.ServeHTTP(w, r)
			return
		}

		// If ?preview=true, we generate a new confirmation token and put it in the request's context
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

		fmt.Println(body.ConfirmationToken)

		if body.ConfirmationToken != acceptedToken {
			http.Error(w, "Confirmation token is invalid", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
