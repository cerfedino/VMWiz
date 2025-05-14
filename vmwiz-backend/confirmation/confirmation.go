package confirmation

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

// Package responsible of generating and verifying confirmation strings for the frontend
// e.g "Type XYZ to confirm"

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const ConfirmationTokenContextField string = "confirmationToken"

// randomString generates a random string of length n.
func randomString(n int) (string, error) {
	result := make([]byte, n)
	alphaLen := big.NewInt(int64(len(letters)))
	for i := range result {
		num, err := rand.Int(rand.Reader, alphaLen)
		if err != nil {
			return "", err
		}
		result[i] = letters[num.Int64()]
	}
	return string(result), nil
}

func VerifyToken(token string) (bool, error) {
	exists, err := storage.DB.ConfirmationPromptTokenExists(token)
	if err != nil {
		return false, fmt.Errorf("Error verifying token: %v", err.Error())
	}
	if !exists {
		return false, nil
	}

	err = storage.DB.ConfirmationPromptTokenSetUsed(token)
	if err != nil {
		return false, fmt.Errorf("Error verifying token: %v", err.Error())
	}

	return true, nil
}

func NewToken() (*string, error) {
	var res *string
	for {
		token, err := randomString(10)
		if err != nil {
			return nil, fmt.Errorf("Error creating confirmation token: %v", err.Error())
		}
		exists, err := storage.DB.ConfirmationPromptTokenExists(token)
		if err != nil {
			return nil, fmt.Errorf("Error creating confirmation token: %v", err.Error())
		}
		if exists {
			continue
		}

		inserted, err := storage.DB.ConfirmationPromptTokenStore(token)
		if err != nil {
			return nil, fmt.Errorf("Error creating confirmation token: %v", err.Error())
		}
		res = inserted
		break
	}
	return res, nil
}

func ConfirmMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			next.ServeHTTP(w, r)
		}

		// If ?preview=true, we generate a new confirmation token and put it in the request's context
		if r.URL.Query().Get("preview") == "true" {
			token, err := NewToken()
			if err != nil {
				msg := fmt.Sprintf("Error in ConfirmMiddleware: %v", err.Error())
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
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

		verified, err := VerifyToken(body.ConfirmationToken)
		if err != nil {
			msg := fmt.Sprintf("Error in ConfirmMiddleware: %v", err.Error())
			log.Printf(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		if !verified {
			http.Error(w, "Confirmation token is invalid", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
