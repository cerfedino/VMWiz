package router

import (
	"encoding/json"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"github.com/gorilla/mux"
)

// Routes under /api/auth/*

func addAllAuthRoutes(r *mux.Router) {

	// Authentication routes
	r.Methods("GET").Path("/api/auth/start").HandlerFunc(auth.StartKeycloakAuthFlow)

	r.Methods("GET").Path("/api/auth/callback").HandlerFunc(auth.HandleKeycloakCallback)

	// Returns the authenticated user's details
	r.Methods("GET").Path("/api/auth/whoami").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := r.Context().Value("user").(auth.KeycloakUser)
		if !ok {
			log.Println("Failed to get user from context in /api/auth/whoami")
			http.Error(w, "Failed to get user", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})))
}
