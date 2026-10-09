package router

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/ratelimit"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/vmrequest/{closure,close,reopen,waitlist}

var waitlistEmailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}$`)

func addRequestClosureRoutes(r *mux.Router) {

	r.Methods("GET").Path("/api/vmrequest/closure").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closure, err := storage.DB.CurrentRequestClosure(r.Context())
		if err != nil {
			log.Printf("Failed to get VM request closure: %v", err)
			http.Error(w, "Failed to get VM request closure", http.StatusInternalServerError)
			return
		}

		type response struct {
			Closed   bool      `json:"closed"`
			Reason   string    `json:"reason,omitempty"`
			ClosedAt time.Time `json:"closedAt,omitzero"`
		}
		resp := response{Closed: closure != nil}
		if closure != nil {
			resp.Reason = closure.Reason
			resp.ClosedAt = closure.ClosedAt
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))

	r.Methods("POST").Path("/api/vmrequest/close").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("close", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Reason string `json:"reason"`
		}
		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		body.Reason = strings.TrimSpace(body.Reason)
		if body.Reason == "" {
			http.Error(w, "A reason is required", http.StatusBadRequest)
			return
		}

		closure, err := storage.DB.CurrentRequestClosure(r.Context())
		if err != nil {
			log.Printf("Failed to get VM request closure: %v", err)
			http.Error(w, "Failed to get VM request closure", http.StatusInternalServerError)
			return
		}
		if closure != nil {
			http.Error(w, "VM requests are already closed", http.StatusBadRequest)
			return
		}

		_, err = storage.DB.CreateRequestClosure(r.Context(), body.Reason)
		if err != nil {
			log.Printf("Failed to close VM requests: %v", err)
			http.Error(w, "Failed to close VM requests", http.StatusInternalServerError)
			return
		}
		log.Printf("VM requests closed: %s", body.Reason)
	}))))

	r.Methods("POST").Path("/api/vmrequest/reopen").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("reopen", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reopened, err := storage.DB.ReopenRequests(r.Context())
		if err != nil {
			log.Printf("Failed to reopen VM requests: %v", err)
			http.Error(w, "Failed to reopen VM requests", http.StatusInternalServerError)
			return
		}
		if reopened == 0 {
			http.Error(w, "VM requests are not closed", http.StatusBadRequest)
			return
		}
		log.Println("VM requests reopened")
	}))))

	r.Methods("POST").Path("/api/vmrequest/waitlist").Handler(ratelimit.PerIP(20*time.Second, 1, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Email string `json:"email"`
		}
		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		email := strings.ToLower(strings.TrimSpace(body.Email))

		// Same shape as the form validation errors, so the frontend shows it inline.
		if !waitlistEmailRegexp.MatchString(email) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"email": "Must be a valid email address"})
			return
		}

		closure, err := storage.DB.CurrentRequestClosure(r.Context())
		if err != nil {
			log.Printf("Failed to get VM request closure: %v", err)
			http.Error(w, "Failed to get VM request closure", http.StatusInternalServerError)
			return
		}
		if closure == nil {
			http.Error(w, "VM requests are open, you can submit a request directly", http.StatusBadRequest)
			return
		}

		err = storage.DB.CreateRequestWaitlistEntry(r.Context(), storage.CreateRequestWaitlistEntryParams{
			ClosureID: closure.ID,
			Email:     email,
		})
		if err != nil {
			log.Printf("Failed to add %s to the VM request waitlist: %v", email, err)
			http.Error(w, "Failed to join the waitlist", http.StatusInternalServerError)
			return
		}
	})))

	r.Methods("GET").Path("/api/vmrequest/waitlist").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closures, err := storage.DB.ListRequestClosures(r.Context())
		if err != nil {
			log.Printf("Failed to list VM request closures: %v", err)
			http.Error(w, "Failed to list VM request closures", http.StatusInternalServerError)
			return
		}
		entries, err := storage.DB.ListRequestWaitlist(r.Context())
		if err != nil {
			log.Printf("Failed to list the VM request waitlist: %v", err)
			http.Error(w, "Failed to list the VM request waitlist", http.StatusInternalServerError)
			return
		}

		type waiter struct {
			Email     string    `json:"email"`
			CreatedAt time.Time `json:"createdAt"`
		}
		type closureResp struct {
			ID         int64      `json:"id"`
			ClosedAt   time.Time  `json:"closedAt"`
			ReopenedAt *time.Time `json:"reopenedAt"`
			Reason     string     `json:"reason"`
			Waitlist   []waiter   `json:"waitlist"`
		}
		waitersByClosure := map[int64][]waiter{}
		for _, e := range entries {
			waitersByClosure[e.ClosureID] = append(waitersByClosure[e.ClosureID], waiter{Email: e.Email, CreatedAt: e.CreatedAt})
		}
		out := make([]closureResp, 0, len(closures))
		for _, c := range closures {
			resp := closureResp{
				ID:       c.ID,
				ClosedAt: c.ClosedAt,
				Reason:   c.Reason,
				Waitlist: waitersByClosure[c.ID],
			}
			if resp.Waitlist == nil {
				resp.Waitlist = []waiter{}
			}
			if c.ReopenedAt.Valid {
				resp.ReopenedAt = &c.ReopenedAt.Time
			}
			out = append(out, resp)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(out)
	})))
}
