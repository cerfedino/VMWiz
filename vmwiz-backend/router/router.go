package router

import (
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

	addVMRequestRoutes(r)

	addAllVMRoutes(r)

	addAllDNSRoutes(r)

	addAllPollRoutes(r)

	addAllAuthRoutes(r)

	return r
}
