package router

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type ErrorBundle struct {
	Err      error
	UserMsg  string
	HttpCode int
}

func SimpleError(err error, msg string) *ErrorBundle {
	log.Printf("%s: %v\n", msg, err)
	return &ErrorBundle{
		Err:      err,
		UserMsg:  msg,
		HttpCode: 500,
	}
}

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
