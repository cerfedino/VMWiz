package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/actionlog"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/vm/*

func addAllVMRoutes(r *mux.Router) {

	r.Methods("POST").Path("/api/vm/deleteByName").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Name      string `json:"vmName"`
			DeleteDNS bool   `json:"deleteDNS"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		uuid, err := storage.DB.ActionLogCreate()
		if err != nil {
			log.Printf("Error creating action log: %v", err)
			http.Error(w, "Failed to create action log", http.StatusInternalServerError)
			return
		}

		vms, err := proxmox.GetAllClusterVMsByName(body.Name)
		if err != nil {
			actionlog.Printf(uuid, "Error getting VM by name: %v", err)
			http.Error(w, "Failed to get VM by name", http.StatusInternalServerError)
			return
		}

		if len(*vms) == 0 {
			actionlog.Printf(uuid, "No VM found with name %s across cluster", body.Name)
			http.Error(w, "No VM found with the given name across cluster", http.StatusNotFound)
			return
		}

		actionlog.Printf(uuid, "Found %v VM(s) across the cluster with name '%v'", len(*vms), body.Name)

		var errors []string
		for idx, vm := range *vms {
			errprefix := fmt.Sprintf("[VM %v/%v]", idx, len(*vms))
			err = proxmox.ForceStopNodeVM(uuid, vm.Node, vm.Vmid)
			if err != nil {
				errmsg := fmt.Sprintf("%v Failed to stop VM %v", errprefix, vm.Id)
				actionlog.Println(uuid, errmsg)
				errors = append(errors, errmsg)
				continue
			}

			err = proxmox.DeleteNodeVM(uuid, vm.Node, vm.Vmid, true, true, false)
			if err != nil {
				errmsg := fmt.Sprintf("%v Failed to delete VM %v", errprefix, vm.Id)
				actionlog.Println(uuid, errmsg)
				errors = append(errors, errmsg)
				continue
			}

			// delete netcenter entry
			if body.DeleteDNS {
				err = netcenter.DeleteDNSEntryByHostname(uuid, vm.Name)
				if err != nil {
					errmsg := fmt.Sprintf("%v Failed to delete DMS entry for VM %v", errprefix, vm.Id)
					actionlog.Println(uuid, errmsg)
					errors = append(errors, errmsg)
					continue
				}
			}
		}

		if len(errors) > 0 {
			http.Error(w, fmt.Sprintf("Errors while deleting some VMs: \n%v", strings.Join(errors, "\n")), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))
}
