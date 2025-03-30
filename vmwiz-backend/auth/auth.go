package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/coreos/go-oidc"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var provider *oidc.Provider
var oauth2Config oauth2.Config
var verifier *oidc.IDTokenVerifier
var ctx context.Context

type KeycloakUser struct {
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

func Init() {
	ctx = context.Background()

	newprovider, err := oidc.NewProvider(ctx, os.Getenv("KEYCLOAK_ISSUER_URL"))
	if err != nil {
		log.Fatalf("Failed to create provider: %v", err)
	}

	provider = newprovider

	verifier = provider.Verifier(&oidc.Config{ClientID: os.Getenv("KEYCLOAK_CLIENT_ID")})

	// Configure an OpenID Connect aware OAuth2 client.
	oauth2Config = oauth2.Config{
		ClientID:     os.Getenv("KEYCLOAK_CLIENT_ID"),
		ClientSecret: os.Getenv("KEYCLOAK_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("VMWIZ_SCHEME") + "://" + os.Getenv("VMWIZ_HOSTNAME") + "/api/auth/callback",

		// Discovery returns the OAuth2 endpoints.
		Endpoint: provider.Endpoint(),

		// "openid" is a required scope for OpenID Connect flows.
		Scopes: []string{oidc.ScopeOpenID, "profile", "roles"},
	}
}

func setCookie(w http.ResponseWriter, r *http.Request, name, value string) {
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
		tokenCookie, err := r.Cookie("auth_token")
		if err != nil {
			fmt.Println("Error getting token cookie:", err)
			http.Redirect(w, r, "/api/auth/start", http.StatusFound)
			return
		}
		if tokenCookie == nil {
			fmt.Println("Token cookie is nil")
			http.Redirect(w, r, "/api/auth/start", http.StatusFound)
			return
		}

		// Verify the token using your oidc verifier.
		idToken, err := verifier.Verify(ctx, tokenCookie.Value)
		if err != nil {
			fmt.Println("Error verifying token:", err)
			http.Redirect(w, r, "/api/auth/start", http.StatusFound)
			return
		}

		// Extract claims. You can decode them into a custom struct or a map.
		var claims KeycloakUser
		err = idToken.Claims(&claims)
		fmt.Println(claims)

		// Attach claims to the request context.
		ctxWithClaims := context.WithValue(r.Context(), "user", claims)
		next.ServeHTTP(w, r.WithContext(ctxWithClaims))
	})
}

func RedirectToKeycloak(w http.ResponseWriter, r *http.Request) {
	uuid, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("Failed to generate UUID: %v", err)
	}
	state := uuid.String()

	setCookie(w, r, "session_state", state)

	http.Redirect(w, r, oauth2Config.AuthCodeURL(state), http.StatusFound)
}

func HandleKeycloakCallback(w http.ResponseWriter, r *http.Request) {
	// Verify state and errors.
	state, err := r.Cookie("session_state")
	if err != nil {
		http.Error(w, "state not found", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("state") != state.Value {
		http.Error(w, fmt.Sprintf("state did not match (%v - %v)", r.URL.Query().Get("state"), state.Value), http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		http.Error(w, fmt.Errorf("Failed to exchange token: %v\nCode: %v", err.Error(), code).Error(), http.StatusBadRequest)
		return
	}

	if oauth2.StaticTokenSource(oauth2Token) == nil {
		http.Error(w, "oauth2.StaticTokenSource is nil", http.StatusInternalServerError)
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

	setCookie(w, r, "auth_token", oauth2Token.Extra("id_token").(string))

	http.Redirect(w, r, "/console", http.StatusFound)
}
