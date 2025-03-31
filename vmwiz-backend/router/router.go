package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/form"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/netcenter"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

func Router() *mux.Router {
	r := mux.NewRouter()

	// Log all requests to console
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
			next.ServeHTTP(w, r)
		})
	})

	r.Methods("POST").Path("/api/vmrequest").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var f form.Form
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			fmt.Println(err.Error())
			http.Error(w, "Form body parsing error", http.StatusInternalServerError)
			return
		}

		log.Printf("Received: %+v\n", f)
		// Validating the received form data
		validation_data, fail := f.Validate()
		if fail {
			resp, _ := json.Marshal(validation_data)
			w.WriteHeader(http.StatusBadRequest)
			w.Write(resp)
			return
		}

		w.WriteHeader(http.StatusOK)

		err = notifier.NotifyVMRequest(f)
		if err != nil {
			log.Printf("Failed to notify VM request: %v", err)
		}

		storage.DB.StoreVMRequest(&f)
	}))

	r.Methods("GET").Path("/api/vmoptions").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, _ := json.Marshal(form.ALLOWED_VALUES)
		w.Write(resp)
	}))

	r.Methods("GET").Path("/api/requests").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vmRequests, err := storage.DB.GetAllVMRequests()
		if err != nil {
			log.Printf("Failed to get VM requests: %v", err)
			http.Error(w, "Failed to get VM requests", http.StatusInternalServerError)
			return
		}
		resp, err := json.Marshal(vmRequests)
		if err != nil {
			log.Printf("Failed to marshal VM requests: %v", err)
			http.Error(w, "Failed to marshal VM requests", http.StatusInternalServerError)
			return
		}
		w.Write(resp)
	})))

	r.Methods("POST").Path("/api/requests/accept").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID int `json:"id"`
		}
		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		request, err := storage.DB.GetVMRequest(int64(body.ID))
		if err != nil {
			log.Printf("Error getting VM request: %v", err)
			http.Error(w, "Failed to fetch VM request", http.StatusInternalServerError)
			return
		}

		opts := request.ToVMOptions()
		opts.Tags = append(opts.Tags, "created-by-vmwiz")

		storage.DB.UpdateVMRequestStatus(int64(body.ID), storage.STATUS_APPROVED)
		_, err = proxmox.CreateVM(*opts)
		if err != nil {
			log.Printf("Error creating VM: %v", err)
			http.Error(w, "Failed to create VM", http.StatusInternalServerError)
			return
		}

	})))

	r.Methods("POST").Path("/api/requests/reject").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID int `json:"id"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		err = storage.DB.UpdateVMRequestStatus(int64(body.ID), storage.STATUS_REJECTED)
		if err != nil {
			log.Printf("Error updating VM request status: %v", err)
			http.Error(w, "Failed to update VM request status", http.StatusInternalServerError)
			return
		}
	})))

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
		err = proxmox.DeleteNodeVM("comp-epyc-lee-3.sos.ethz.ch", vm.Vmid, true, true, false)
		if err != nil {
			log.Printf("Error deleting VM: %v", err)
			http.Error(w, "Failed to delete VM", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	})))

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

	// Authentication routes
	r.Methods("GET").Path("/api/auth/start").HandlerFunc(auth.RedirectToKeycloak)
	r.Methods("GET").Path("/api/auth/callback").HandlerFunc(auth.HandleKeycloakCallback)

	// r.Methods("POST").Path("/api/auth/login").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	// Retrieve soseth_username and soseth_password from the request body
	// 	var credentials struct {
	// 		Username string `json:"soseth_username"`
	// 		Password string `json:"soseth_password"`
	// 	}

	// 	err := json.NewDecoder(r.Body).Decode(&credentials)
	// 	if err != nil {
	// 		log.Printf("Error decoding JSON: %v", err)
	// 		http.Error(w, "Invalid request payload", http.StatusBadRequest)
	// 		return
	// 	}

	// 	// Authenticate the user
	// 	user, err := auth.Authenticate(credentials.Username, credentials.Password)
	// 	if err != nil {
	// 		log.Printf("Authentication error: %v", err)
	// 		http.Error(w, "Authentication failed", http.StatusUnauthorized)
	// 		return
	// 	}

	// 	auth.SetAuthHeaders(w, *user)
	// 	// Write token in body
	// 	resp, err := json.Marshal(user)

	// }))

	return r
}
