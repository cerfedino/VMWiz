package router

import (
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"github.com/gorilla/mux"
)

// Routes under /api/auth/*

func addAllAuthRoutes(r *mux.Router) {

	// Authentication routes
	r.Methods("GET").Path("/api/auth/start").HandlerFunc(auth.StartKeycloakAuthFlow)

	r.Methods("GET").Path("/api/auth/callback").HandlerFunc(auth.HandleKeycloakCallback)
}
