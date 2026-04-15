package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"github.com/gorilla/mux"
)

// Routes under /api/vm/*

func addAllVMRoutes(r *mux.Router) {

	r.Methods("POST").Path("/api/vm/deleteByName").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("delete vm", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		vms, err := proxmox.GetAllClusterVMsByName(body.Name)
		if err != nil {
			log.Printf("Error getting VM by name: %v", err)
			http.Error(w, "Failed to get VM by name", http.StatusInternalServerError)
			return
		}

		if len(*vms) == 0 {
			log.Printf("No VM found with name %s across cluster", body.Name)
			http.Error(w, "No VM found with the given name across cluster", http.StatusNotFound)
			return
		}

		opLogger := &OperationLogger{OperationID: fmt.Sprintf("vmdelete-%s", body.Name)}
		opLogger.Printf("Found %v VM(s) across the cluster with name '%v'", len(*vms), body.Name)

		go func() {
			var errors []string
			for idx, vm := range *vms {
				errprefix := fmt.Sprintf("[VM %v/%v]", idx+1, len(*vms))
				opLogger.Printf("%v Stopping VM %v", errprefix, vm.Id)
				err = proxmox.ForceStopNodeVM(vm.Node, vm.Vmid)
				if err != nil {
					errmsg := fmt.Sprintf("%v Failed to stop VM %v: %v", errprefix, vm.Id, err)
					opLogger.Println(errmsg)
					errors = append(errors, errmsg)
					continue
				}

				opLogger.Printf("%v Deleting VM %v", errprefix, vm.Id)
				err = proxmox.DeleteNodeVM(vm.Node, vm.Vmid, true, true, false)
				if err != nil {
					errmsg := fmt.Sprintf("%v Failed to delete VM %v: %v", errprefix, vm.Id, err)
					opLogger.Println(errmsg)
					errors = append(errors, errmsg)
					continue
				}

				// delete netcenter entry
				if body.DeleteDNS {
					opLogger.Printf("%v Deleting DNS entry for VM %v", errprefix, vm.Id)
					err = netcenter.DeleteDNSEntryByHostname(vm.Name)
					if err != nil {
						errmsg := fmt.Sprintf("%v Failed to delete DNS entry for VM %v: %v", errprefix, vm.Id, err)
						opLogger.Println(errmsg)
						errors = append(errors, errmsg)
						continue
					}
				}
				opLogger.Printf("%v Successfully processed VM %v", errprefix, vm.Id)
			}

			if len(errors) > 0 {
				opLogger.Printf("Errors while deleting some VMs: \n%v", strings.Join(errors, "\n"))
			} else {
				opLogger.Println("VM deletion completed successfully.")
			}
		}()

		w.WriteHeader(http.StatusOK)
	}))))
}
