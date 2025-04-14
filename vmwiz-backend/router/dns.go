package router

import (
	"encoding/json"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"github.com/gorilla/mux"
)

// Routes under /api/dns/*

func addAllDNSRoutes(r *mux.Router) {

	r.Methods("POST").Path("/api/dns/deleteByHostname").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Hostname string `json:"hostname"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		err = netcenter.DeleteDNSEntryByHostname(body.Hostname)
		if err != nil {
			log.Printf("Error deleting DNS entry: %v", err)
			http.Error(w, "Failed to delete DNS entry", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))

}
