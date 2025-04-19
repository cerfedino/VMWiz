package router

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/survey"
	"github.com/gorilla/mux"
)

// Routes under /api/usagesurvey/*

func addAllPollRoutes(r *mux.Router) {

	r.Methods("GET").Path("/api/usagesurvey/").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type response struct {
			Surveys []int64 `json:"surveyIds"`
		}

		surveys, err := storage.DB.SurveyGetAllIDs()
		if err != nil {
			log.Printf("Error getting all surveys: %v", err)
			http.Error(w, "Failed to get all surveys", http.StatusInternalServerError)
			return
		}

		resp := response{
			Surveys: surveys,
		}
		respJSON, err := json.Marshal(resp)
		if err != nil {
			log.Printf("Error marshalling surveys: %v", err)
			http.Error(w, "Failed to marshal surveys", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(respJSON)
	})))

	r.Methods("GET").Path("/api/usagesurvey/info").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get id from query
		query := r.URL.Query()
		surveyIdstr := query.Get("surveyId")
		if surveyIdstr == "" {
			log.Println("No surveyId provided")
			http.Error(w, "No surveyId provided", http.StatusBadRequest)
			return
		}
		surveyId, err := strconv.ParseInt(surveyIdstr, 10, 64)
		if err != nil {
			log.Printf("Error casting surveyId to int: %v", err)
			http.Error(w, "Invalid surveyId provided", http.StatusBadRequest)
			return
		}

		survey, err := storage.DB.SurveyGetById(surveyId)
		if err != nil {
			log.Printf("Error retrieving survey from DB: %v", err)
			http.Error(w, "Failed to retrieve survey", http.StatusInternalServerError)
			return
		}

		unsent, err := storage.DB.SurveyEmailCountNotSent(survey.Id)
		if err != nil {
			log.Printf("Error getting unsent emails: %v", err)
			http.Error(w, "Failed to get unsent emails", http.StatusInternalServerError)
			return
		}
		positive, err := storage.DB.SurveyEmailCountPositive(survey.Id)
		if err != nil {
			log.Printf("Error getting positive emails: %v", err)
			http.Error(w, "Failed to get positive emails", http.StatusInternalServerError)
			return
		}
		negative, err := storage.DB.SurveyEmailCountNegative(survey.Id)
		if err != nil {
			log.Printf("Error getting negative emails: %v", err)
			http.Error(w, "Failed to get negative emails", http.StatusInternalServerError)
			return
		}
		notResponded, err := storage.DB.SurveyEmailCountNotResponded(survey.Id)
		if err != nil {
			log.Printf("Error getting not responded emails: %v", err)
			http.Error(w, "Failed to get not responded emails", http.StatusInternalServerError)
			return
		}

		type response struct {
			SurveyId     int64     `json:"surveyId"`
			Date         time.Time `json:"date"`
			Positive     int       `json:"positive"`
			Negative     int       `json:"negative"`
			NotResponded int       `json:"not_responded"`
			Not_Sent     int       `json:"not_sent"`
		}

		resp := response{
			SurveyId:     survey.Id,
			Date:         survey.Date,
			Not_Sent:     *unsent,
			Positive:     *positive,
			Negative:     *negative,
			NotResponded: *notResponded,
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

	r.Methods("GET").Path("/api/usagesurvey/start").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	r.Methods("POST").Path("/api/usagesurvey/set").Subrouter().NewRoute().Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		exists, err := storage.DB.SurveyEmailExistsByUUID(body.ID)
		if err != nil {
			log.Printf("Error checking if survey response exists: %v", err)
			http.Error(w, "Failed to check if survey response exists", http.StatusInternalServerError)
			return
		}
		if !exists {
			log.Println("Survey response does not exist")
			http.Error(w, "Invalid survey ID", http.StatusNotFound)
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

	r.Methods("POST").Path("/api/usagesurvey/resend").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID int64 `json:"id"`
		}
		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		err = survey.SendVMUsageSurvey(body.ID)
		if err != nil {
			log.Printf("Error sending survey: %v", err)
			http.Error(w, "Failed to send survey", http.StatusInternalServerError)
			return
		}
		//notifier how
	})))

	r.Methods("GET").Path("/api/usagesurvey/responses/positive").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		responses, err := storage.DB.SurveyEmailPositive(idInt)
		if err != nil {
			log.Printf("Error getting survey responses: %v", err)
			http.Error(w, "Failed to get survey responses", http.StatusInternalServerError)
			return
		}
		resp, _ := json.Marshal(responses)
		w.Write(resp)
	})))

	r.Methods("GET").Path("/api/usagesurvey/responses/negative").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	r.Methods("GET").Path("/api/usagesurvey/responses/notsent").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		responses, err := storage.DB.SurveyEmailNotSent(idInt)
		if err != nil {
			log.Printf("Error getting unsent emails: %v", err)
			http.Error(w, "Failed to get unsent emails", http.StatusInternalServerError)
			return
		}
		resp, _ := json.Marshal(responses)
		w.Write(resp)
	})))

	r.Methods("GET").Path("/api/usagesurvey/responses/none").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
