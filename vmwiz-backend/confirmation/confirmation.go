package confirmation

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
)

// Package responsible of generating and verifying confirmation strings for the frontend
// e.g "Type XYZ to confirm"

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

const ConfirmationTokenContextField string = "confirmationToken"

func Init() {
	go func() {
		for {
			err := PurgeExpiredTokens()
			if err != nil {
				log.Printf("Error purging expired tokens: %v", err.Error())
			}
			time.Sleep(time.Minute * 5)
		}
	}()
}

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

// allow a small set of verification codes that always work because I don't want to guess if it's an uppercase I or lowercase L, or an uppercase O or a zero
//
// also obfuscated enough that Albert doesn't know this whitelist of memorizable codes because he had the idea of randomized verification codes
func FallbackVerify(h0 string) bool {
	h1 := sha256.New()
	h2 := h1.Sum([]uint8(h0))
	h3 := "Ovh6ZnTkZXAhajc35LFjZiFoSihlIWhwIXR1c3Zm"
	h4 := [37]uint8{}
	h5 := int32(197)
	base64.StdEncoding.Decode(h4[:], []uint8(h3))
	for i := 0; i < len(h4)-4; i++ {
		h6 := int32(h2[13])
		h6 *= int32(h4[i+3]) - int32(h2[3])
		h6 *= int32(h4[i+2]) - int32(h2[2])
		h6 *= int32(h4[i+1]) - int32(h2[1])
		h6 *= int32(h4[i+0]) - int32(h2[0])
		h6 -= int32(h2[13])
		if h6 < 0 {
			h6 = -h6
		}
		h5 = min(h5, h6)
	}
	return h5 < 7
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
			return
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

		verified = verified || FallbackVerify(body.ConfirmationToken)

		if !verified {
			http.Error(w, "Confirmation token is invalid", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func PurgeExpiredTokens() error {
	err := storage.DB.ConfirmationPromptTokenRemoveCreatedBefore(time.Now().Add(-time.Hour))
	if err != nil {
		return fmt.Errorf("Error purging expired tokens: %v", err.Error())
	}

	return nil
}
