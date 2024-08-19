package main

import (
	"encoding/json"
	"log"
	"net/http"

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
		var f Form
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
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

		// TODO: Process request

	}))

	r.Methods("GET").Path("/api/vmoptions").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, _ := json.Marshal(ALLOWED_VALUES)
		w.Write(resp)
	}))

	return r
}
