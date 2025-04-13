package router

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/survey"
	"github.com/gorilla/mux"
)

// Routes under /api/poll/*

func addAllPollRoutes(r *mux.Router) {

	r.Methods("GET").Path("/api/poll/start").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		surveyId, err := survey.CreateVMUsageSurvey([]string{"vsos"})
		if err != nil {
			log.Printf("Error sending survey: %v", err)
			http.Error(w, "Failed to send survey", http.StatusInternalServerError)
			return
		}

		// Marshal a struct containing surveyId field
		type response struct {
			SurveyID int64 `json:"surveyId"`
		}
		resp := response{
			SurveyID: *surveyId,
		}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshalling response: %v", err)
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(respJSON)
	})))

	r.Methods("POST").Path("/api/poll/set").Subrouter().NewRoute().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID   string `json:"id"`
			Keep bool   `json:"keep"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		err = storage.DB.SurveyEmailUpdateResponse(body.ID, body.Keep)
		if err != nil {
			log.Printf("Error setting survey response: %v", err)
			http.Error(w, "Failed to set survey response", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	r.Methods("GET").Path("/api/poll/lastsurvey").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		surveys, err := storage.DB.SurveyGetLastId()
		if err != nil {
			log.Printf("Error getting last survey: %v", err)
			http.Error(w, "Failed to get last survey", http.StatusInternalServerError)
			return
		}
		resp, err := json.Marshal(surveys)
		if err != nil {
			log.Printf("Error marshalling last survey: %v", err)
			http.Error(w, "Failed to marshal last survey", http.StatusInternalServerError)
			return
		}
		w.Write(resp)
	})))

	r.Methods("GET").Path("/api/poll/responses/negative").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get id from query
		query := r.URL.Query()
		id := query.Get("id")
		if id == "" {
			log.Println("No id provided")
			http.Error(w, "No id provided", http.StatusBadRequest)
			return
		}
		// cast id to int
		idInt, err := strconv.Atoi(id)
		if err != nil {
			log.Printf("Error casting id to int: %v", err)
			http.Error(w, "Invalid id provided", http.StatusBadRequest)
			return
		}
		responses, err := storage.DB.SurveyEmailNegative(idInt)
		if err != nil {
			log.Printf("Error getting survey responses: %v", err)
			http.Error(w, "Failed to get survey responses", http.StatusInternalServerError)
			return
		}
		resp, _ := json.Marshal(responses)
		w.Write(resp)
	})))

	r.Methods("GET").Path("/api/poll/responses/none").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get id from query
		query := r.URL.Query()
		id := query.Get("id")
		if id == "" {
			log.Println("No id provided")
			http.Error(w, "No id provided", http.StatusBadRequest)
			return
		}
		// cast id to int
		idInt, err := strconv.Atoi(id)
		if err != nil {
			log.Printf("Error casting id to int: %v", err)
			http.Error(w, "Invalid id provided", http.StatusBadRequest)
			return
		}
		responses, err := storage.DB.SurveyEmailNotResponded(idInt)
		if err != nil {
			log.Printf("Error getting survey responses: %v", err)
			http.Error(w, "Failed to get survey responses", http.StatusInternalServerError)
			return
		}
		resp, _ := json.Marshal(responses)
		w.Write(resp)
	})))
}
