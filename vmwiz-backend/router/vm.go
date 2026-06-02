package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
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

		// Create a new logging sub-scope
		ctx, lg, finish := logger.Nest(context.Background(), fmt.Sprintf("Delete VM %s", body.Name))
		w.Header().Set("X-Log-Scope-Id", lg.ScopeID())
		w.WriteHeader(http.StatusAccepted)

		go func() {
			var failed error
			// Make sure the logs for this scope are set to be finished once the function is over
			defer func() { finish(failed) }()

			lg.Infof("Found %v VM(s) across the cluster with name '%v'", len(*vms), body.Name)
			var errors []string
			for idx, vm := range *vms {
				errprefix := fmt.Sprintf("[VM %v/%v]", idx+1, len(*vms))

				lg.Infof("%v Stopping VM %v", errprefix, vm.Id)
				if err := proxmox.ForceStopNodeVM(ctx, vm.Node, vm.Vmid); err != nil {
					lg.Errorf("%v Failed to stop VM %v: %v", errprefix, vm.Id, err)
					errors = append(errors, fmt.Sprintf("%v Failed to stop VM %v", errprefix, vm.Id))
					continue
				}

				lg.Infof("%v Deleting VM %v", errprefix, vm.Id)
				if err := proxmox.DeleteNodeVM(ctx, vm.Node, vm.Vmid, true, true, false); err != nil {
					lg.Errorf("%v Failed to delete VM %v: %v", errprefix, vm.Id, err)
					errors = append(errors, fmt.Sprintf("%v Failed to delete VM %v", errprefix, vm.Id))
					continue
				}

				if body.DeleteDNS {
					lg.Infof("%v Deleting DNS entry for VM %v", errprefix, vm.Id)
					if err := netcenter.DeleteDNSEntryByHostname(ctx, vm.Name); err != nil {
						lg.Errorf("%v Failed to delete DNS entry for VM %v: %v", errprefix, vm.Id, err)
						errors = append(errors, fmt.Sprintf("%v Failed to delete DNS entry for VM %v", errprefix, vm.Id))
						continue
					}
				}
			}

			if len(errors) > 0 {
				failed = fmt.Errorf("errors while deleting some VMs:\n%s", strings.Join(errors, "\n"))
			} else {
				lg.Info("VM deletion completed successfully")
			}
		}()
	}))))

	r.Methods("GET").Path("/api/vm/ipv4free").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		free, err := netcenter.GetFreeIPv4sInSubnet(netcenter.VM_SUBNET.V4net)
		if err != nil {
			log.Printf("Failed getting free IPs: %v", err)
			http.Error(w, "Failed to get free IPs", http.StatusInternalServerError)
			return
		}

		type Resp struct {
			Count int `json:"count"`
		}
		resp, _ := json.Marshal(Resp{Count: len(*free)})
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})))
}
