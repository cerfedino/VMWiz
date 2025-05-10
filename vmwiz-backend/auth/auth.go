package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var provider *oidc.Provider
var oauth2Config oauth2.Config
var verifier *oidc.IDTokenVerifier
var ctx context.Context
var tokenSourceMap = make(map[string]oauth2.TokenSource)

type KeycloakUser struct {
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

func Init() {
	ctx = context.Background()

	newprovider, err := oidc.NewProvider(ctx, config.AppConfig.KEYCLOAK_ISSUER_URL)
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	provider = newprovider

	verifier = provider.Verifier(&oidc.Config{ClientID: config.AppConfig.KEYCLOAK_CLIENT_ID})

	// Configure an OpenID Connect aware OAuth2 client.
	oauth2Config = oauth2.Config{
		ClientID:     config.AppConfig.KEYCLOAK_CLIENT_ID,
		ClientSecret: config.AppConfig.KEYCLOAK_CLIENT_SECRET,
		RedirectURL:  config.AppConfig.VMWIZ_SCHEME + "://" + config.AppConfig.VMWIZ_HOSTNAME + "/api/auth/callback",

		Endpoint: provider.Endpoint(),

		Scopes: []string{oidc.ScopeOpenID, "profile", "roles"},
	}
	go func() {
		for {
			time.Sleep(24 * time.Hour)
			// delete all expired tokens
			for k, v := range tokenSourceMap {
				token, err := v.Token()
				if err != nil || token.Expiry.Before(time.Now()) {
					delete(tokenSourceMap, k)
				}
			}
		}
	}()
}

func setCookie(w http.ResponseWriter, r *http.Request, name string, value string) {
	c := &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(time.Hour.Seconds()),
		Secure:   r.TLS != nil,
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, c)
}

// Checks if the user is authenticated or redirects him to the login endpoint instead.
func CheckAuthenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.AppConfig.AUTH_SKIP {
			next.ServeHTTP(w, r)
			return
		}

		tokenCookie, err := r.Cookie("auth_token")

		if err != nil {
			log.Println("Error getting token cookie:", err)
			replyUnauthorized(w)
			return
		}
		if tokenCookie == nil {
			log.Println("Token cookie is nil")
			replyUnauthorized(w)
			return
		}
		// get tokesource from map
		tokenSource, ok := tokenSourceMap[tokenCookie.Value]
		if ok {
			token, err := tokenSource.Token()
			if err == nil {
				// delete the old token
				delete(tokenSourceMap, tokenCookie.Value)
				tokenCookie.Value = token.Extra("id_token").(string)
				setCookie(w, r, "auth_token", tokenCookie.Value)
				tokenSourceMap[tokenCookie.Value] = tokenSource
			}
		}

		// Verify the token
		idToken, err := verifier.Verify(ctx, tokenCookie.Value)
		if err != nil {
			log.Println("Error verifying token:", err)
			replyUnauthorized(w)
			return
		}

		var claims KeycloakUser
		err = idToken.Claims(&claims)
		for _, group := range claims.Groups {
			if len(config.AppConfig.KEYCLOAK_RESTRICT_AUTH_TO_GROUPS) == 0 {
				checkAuthenticatedSuccess(w, r, next, claims)
				return
			}
			for _, allowed_group := range config.AppConfig.KEYCLOAK_RESTRICT_AUTH_TO_GROUPS {
				if strings.TrimPrefix(group, "/") == allowed_group {
					checkAuthenticatedSuccess(w, r, next, claims)
					return
				}
			}

		}
		http.Error(w, "Your user is not allowed to use this route", http.StatusUnauthorized)
		return
	})
}

// Tells the client that he is not authenticated/authorized and instructs him to begin the auth flow
func replyUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	body := RedirectUrlBody{
		RedirectURL: "/api/auth/start",
	}
	respJSON, err := json.Marshal(body)
	if err != nil {
		log.Printf("Error marshalling redirect url: %v", err)
		http.Error(w, "Failed to marshal redirect url", http.StatusInternalServerError)
		return
	}

	http.Error(w, string(respJSON), http.StatusUnauthorized)
}

func checkAuthenticatedSuccess(w http.ResponseWriter, r *http.Request, next http.Handler, claims KeycloakUser) {
	// Attach claims to the request context.
	ctxWithClaims := context.WithValue(r.Context(), "user", claims)
	next.ServeHTTP(w, r.WithContext(ctxWithClaims))
}

func StartKeycloakAuthFlow(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	state := uuid.String()

	setCookie(w, r, "session_state", state)

	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
	return
}

type RedirectUrlBody struct {
	RedirectURL string `json:"redirectUrl"`
}

func HandleKeycloakCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state and errors.
	state, err := r.Cookie("session_state")
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, fmt.Sprintf("state did not match (query: %v - cookie: %v)", r.URL.Query().Get("state"), state.Value), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, fmt.Errorf("Failed to exchange token: %v\nCode: %v", err.Error(), code).Error(), http.StatusBadRequest)
		return
	}

	// Parse and verify token
	userInfo, err := provider.UserInfo(ctx, oauth2.StaticTokenSource(oauth2Token))
	if err != nil {
		http.Error(w, "Failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var claims KeycloakUser
	err = userInfo.Claims(&claims)

	if err != nil {
		http.Error(w, "Failed to get claims: "+err.Error(), http.StatusInternalServerError)
		return
	}
	tokenString := oauth2Token.Extra("id_token").(string)
	ts := oauth2Config.TokenSource(ctx, oauth2Token)
	tokenSourceMap[tokenString] = ts

	setCookie(w, r, "auth_token", tokenString)

	http.Redirect(w, r, "/console", http.StatusFound)
}
