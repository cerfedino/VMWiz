package router

import (
	"fmt"
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

func NotifyVMRequest(req storage.SQLVMRequest) error {
	return notifier.UseNotifier("new_vmrequest", req.ToString())
}

func NotifyVMRequestStatusChanged(req storage.SQLVMRequest) error {
	switch req.RequestStatus {
	case storage.STATUS_ACCEPTED:
		return notifier.UseNotifier("vmrequest_accepted", fmt.Sprintf("Request %v approved !", req.ID))
	case storage.STATUS_REJECTED:
		return notifier.UseNotifier("vmrequest_rejected", fmt.Sprintf("Request %v denied !", req.ID))
	}

	return nil
}
