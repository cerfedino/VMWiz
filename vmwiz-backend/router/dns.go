package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/netcenter"
	"github.com/gorilla/mux"
)

// Routes under /api/dns/*

func addAllDNSRoutes(r *mux.Router) {

	r.Methods("POST").Path("/api/dns/deleteByHostname").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("delete dns", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx, lg, finish := logger.Nest(context.Background(), fmt.Sprintf("Delete DNS for %s", body.Hostname))
		w.Header().Set("X-Log-Scope-Id", lg.ScopeID())
		w.WriteHeader(http.StatusAccepted)

		go func() {
			lg.Infof("Deleting DNS entries for %s", body.Hostname)
			finish(netcenter.DeleteDNSEntryByHostname(ctx, body.Hostname))
		}()
	}))))

}
