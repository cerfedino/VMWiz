package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/form"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
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
		vmRequests, err := storage.DB.GetAllVMsRequest()
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

	r.Methods("GET").Path("/api/auth/start").HandlerFunc(auth.RedirectToKeycloak)
	r.Methods("GET").Path("/api/auth/callback").HandlerFunc(auth.HandleKeycloakCallback)


	return r
}
