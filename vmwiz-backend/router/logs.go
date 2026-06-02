package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

func parseTimeParam(r *http.Request, key string) (*time.Time, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func addLogRoutes(r *mux.Router) {
	// Lists the most recent top-level log scopes (ongoing and completed).
	r.Methods("GET").Path("/api/logs").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		before := req.URL.Query().Get("before")
		limit := 30
		if n, err := strconv.Atoi(req.URL.Query().Get("limit")); err == nil && n > 0 && n <= 100 {
			limit = n
		}

		scopes, err := storage.DB.ListRootScopes(before, limit)
		if err != nil {
			log.Printf("Failed to list log scopes: %v", err)
			http.Error(w, "Failed to list log scopes", http.StatusInternalServerError)
			return
		}

		type scopeResp struct {
			ID        string    `json:"id"`
			Label     string    `json:"label"`
			StartedAt time.Time `json:"startedAt"`
			Ended     bool      `json:"ended"`
			Failed    bool      `json:"failed"`
			Available bool      `json:"available"`
		}
		out := []scopeResp{}
		for _, sc := range scopes {
			out = append(out, scopeResp{
				ID:        sc.ID,
				Label:     sc.Label,
				StartedAt: sc.StartedAt,
				Ended:     sc.EndedAt.Valid,
				Failed:    sc.Failed,
				Available: logger.LogFileExists(sc.ID),
			})
		}

		resp, _ := json.Marshal(out)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})))

	// Reads ALL of the logs from a scope (and optionally its subscopes, and in a specific date range)
	r.Methods("GET").Path("/api/logs/{scope:[A-Za-z0-9_-]+}").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		scope := mux.Vars(req)["scope"]
		includeSubscopes := req.URL.Query().Get("subscopes") == "true"

		from, err := parseTimeParam(req, "from")
		if err != nil {
			http.Error(w, "Invalid 'from' timestamp", http.StatusBadRequest)
			return
		}
		to, err := parseTimeParam(req, "to")
		if err != nil {
			http.Error(w, "Invalid 'to' timestamp", http.StatusBadRequest)
			return
		}

		lines, err := logger.ReadLogs(scope, includeSubscopes, from, to)
		if err != nil {
			log.Printf("Failed to read logs for scope %s: %v", scope, err)
			http.Error(w, "Failed to read logs", http.StatusInternalServerError)
			return
		}

		resp, _ := json.Marshal(lines)
		w.Header().Set("Content-Type", "application/json")
		w.Write(resp)
	})))

	// Streams logs as they happen
	r.Methods("GET").Path("/api/logs/{scope:[A-Za-z0-9_-]+}/stream").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		scope := mux.Vars(req)["scope"]
		includeSubscopes := req.URL.Query().Get("subscopes") == "true"

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}
		_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})

		reader, err := logger.NewLogReader(scope, includeSubscopes)
		if err != nil {
			http.Error(w, "Failed to open logs", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		fmt.Fprint(w, ": connected\n\n")
		flusher.Flush()

		ctx := req.Context()
		// Sends updates every 300ms
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

		for {
			lines, err := reader.Next()
			if err != nil {
				log.Printf("Stream scope %s: %v", scope, err)
				return
			}
			for _, l := range lines {
				b, _ := json.Marshal(l)
				fmt.Fprintf(w, "data: %s\n\n", b)
			}
			if len(lines) > 0 {
				flusher.Flush()
			} else if finished, failed := logger.ScopeFinished(scope); finished {
				// finish() writes its final line before setting ended_at, so
				// drain once more before declaring the stream done.
				final, _ := reader.Next()
				for _, l := range final {
					b, _ := json.Marshal(l)
					fmt.Fprintf(w, "data: %s\n\n", b)
				}
				fmt.Fprintf(w, "event: done\ndata: {\"failed\":%t}\n\n", failed)
				flusher.Flush()
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	})))
}
