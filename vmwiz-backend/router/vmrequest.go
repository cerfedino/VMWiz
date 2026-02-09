package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/form"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/app.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/vmrequest/*

func AcceptVMRequest(id int64) *ErrorBundle {
	request, err := storage.DB.GetVMRequestById(id)

	if err != nil {
		return SimpleError(err, "Error fetching VM request")
	}

	request.RequestStatus = storage.REQUEST_STATUS_ACCEPTED
	err = notifier.NotifyVMRequestStatusChanged(*request, "Creating VM now, it'll take a while ...")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	opts := request.ToVMOptions()
	if request.IsOrganization {
		opts.ResourcePool = config.AppConfig.VM_ORGANIZATION_POOL
	} else {
		opts.ResourcePool = config.AppConfig.VM_PERSONAL_POOL
	}

	err = storage.DB.UpdateVMRequestStatus(id, storage.REQUEST_STATUS_ACCEPTED)
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	_, summary, err := proxmox.CreateVM(*opts)
	if err != nil {
		err2 := notifier.NotifyVMCreationUpdate(fmt.Sprintf("Request %d: Error creating VM:\n%v", id, "```\n"+err.Error()+"\n```"))
		if err2 != nil {
			return SimpleError(err2, "Failed to notify VM creation update")
		}
		return SimpleError(err, "Failed to create VM")
	}

	//send mail to the user
	err = notifier.SendEmail("VSOS VM Creation", []byte(summary.String()), []string{request.Email, config.AppConfig.SMTP_REPLYTO})
	if err != nil {
		return SimpleError(err, "Failed to send email")
	}

	successMsg := fmt.Sprintf("Request %v: VM %s created successfully:\n%s", request.ID, opts.FQDN, "```\n"+summary.String()+"\n```")
	fmt.Println(successMsg)

	err = notifier.NotifyVMCreationUpdate(successMsg)
	if err != nil {
		return SimpleError(err, "Failed to notify VM creation update")
	}

	return nil
}

func RejectVMRequest(id int64) *ErrorBundle {
	// Ensure we didnt accept the request previously
	request, err := storage.DB.GetVMRequestById(id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	if request.RequestStatus == storage.REQUEST_STATUS_ACCEPTED {
		return SimpleError(nil, "Cannot reject an accepted request")
	}

	err = storage.DB.UpdateVMRequestStatus(id, storage.REQUEST_STATUS_REJECTED)
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	request, err = storage.DB.GetVMRequestById(id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	err = notifier.NotifyVMRequestStatusChanged(*request, "")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	fmt.Printf("Rejected VM request %d (%s).\n", id, request.ToVMOptions().FQDN)

	return nil
}

func addVMRequestRoutes(r *mux.Router) {

	// TODO: Rate limit requests
	r.Methods("POST").Path("/api/vmrequest").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var f form.Form
		err := json.NewDecoder(r.Body).Decode(&f)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Form body parsing error", http.StatusInternalServerError)
			return
		}

		// Validating the received form data
		validation_data, fail := f.Validate()
		if fail {
			resp, _ := json.Marshal(validation_data)
			w.WriteHeader(http.StatusForbidden)
			w.Header().Set("Content-Type", "application/json")
			w.Write(resp)
			return
		}

		w.WriteHeader(http.StatusOK)

		id, err := storage.DB.StoreVMRequest(&f)
		if err != nil {
			log.Printf("Failed to store VM request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		req, err := storage.DB.GetVMRequestById(*id)
		if err != nil {
			log.Printf("Failed to get VM request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = notifier.NotifyVMRequest(*req)
		if err != nil {
			log.Printf("Failed to send notification: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

	}))

	r.Methods("GET").Path("/api/vmrequest/options").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp, _ := json.Marshal(form.ALLOWED_VALUES)
		w.Write(resp)
	}))

	r.Methods("GET").Path("/api/vmrequest").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	r.Methods("POST").Path("/api/vmrequest/accept").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if token := r.Context().Value(confirmation.ConfirmationTokenContextField); token != nil {
			type response struct {
				ConfirmationToken string `json:"confirmationToken"`
			}

			resp := response{
				ConfirmationToken: (token.(string)),
			}
			respJSON, err := json.Marshal(resp)
			if err != nil {
				log.Printf("Error marshalling response: %v", err)
				http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(respJSON)
			return
		}

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

		eb := AcceptVMRequest(int64(body.ID))

		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}

	}))))

	r.Methods("POST").Path("/api/vmrequest/reject").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		eb := RejectVMRequest(int64(body.ID))

		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	})))

	r.Methods("POST").Path("/api/vmrequest/edit").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Hostname   string `json:"hostname"`
			ID         int    `json:"id"`
			Cores_cpu  int    `json:"cores_cpu"`
			Ram_gb     int    `json:"ram_gb"`
			Storage_gb int    `json:"storage_gb"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		request, err := storage.DB.GetVMRequestById(int64(body.ID))
		if err != nil {
			log.Printf("Error getting VM request: %v", err)
			http.Error(w, "Failed to fetch VM request", http.StatusInternalServerError)
			return
		}

		if body.Cores_cpu != 0 {
			request.Cores = body.Cores_cpu
		}
		if body.Ram_gb != 0 {
			request.RamGB = body.Ram_gb
		}
		if body.Storage_gb != 0 {
			request.DiskGB = body.Storage_gb
		}
		if body.Hostname != "" {
			request.Hostname = body.Hostname
		}

		err = storage.DB.UpdateVMRequest(*request)
		if err != nil {
			log.Printf("Error updating VM request: %v", err)
			http.Error(w, "Failed to update VM request", http.StatusInternalServerError)
			return
		}

	})))
}
