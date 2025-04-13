package router

import (
	"encoding/json"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"github.com/gorilla/mux"
)

// Routes under /api/vm/*

func addAllVMRoutes(r *mux.Router) {

	r.Methods("POST").Path("/api/vm/deleteByName").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Name string `json:"vmName"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		vms, err := proxmox.GetAllNodeVMsByName("comp-epyc-lee-3", body.Name)
		if err != nil {
			log.Printf("Error getting VM by name: %v", err)
			http.Error(w, "Failed to get VM by name", http.StatusInternalServerError)
			return
		}
		if len(*vms) == 0 {
			log.Printf("No VM found with name: %s", body.Name)
			http.Error(w, "No VM found with the given name", http.StatusNotFound)
			return
		}
		vm := (*vms)[0]

		err = proxmox.ForceStopNodeVM("comp-epyc-lee-3", vm.Vmid)
		if err != nil {
			log.Printf("Error stopping VM: %v", err)
			http.Error(w, "Failed to stop VM", http.StatusInternalServerError)
			return
		}

		err = proxmox.DeleteNodeVM("comp-epyc-lee-3", vm.Vmid, true, true, false)
		if err != nil {
			log.Printf("Error deleting VM: %v", err)
			http.Error(w, "Failed to delete VM", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))
}
