package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/form"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/logger"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/vmrequest/*

// AcceptVMRequest marks a VM request as accepted, creates the VM,
// sends notifications, and emails the requester.
// Returns an ErrorBundle if any step fails.
func AcceptVMRequest(ctx context.Context, id int64) *ErrorBundle {
	request, err := storage.DB.GetVMRequestByID(ctx, id)

	if err != nil {
		return SimpleError(err, "Error fetching VM request")
	}

	request.Requeststatus = storage.REQUEST_STATUS_ACCEPTED
	err = notifier.NotifyVMRequestStatusChanged(ctx, request, "Creating VM now, it'll take a while ...")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	opts := request.ToVMOptions()
	if request.Isorganization {
		opts.ResourcePool = config.AppConfig.VM_ORGANIZATION_POOL
	} else {
		opts.ResourcePool = config.AppConfig.VM_PERSONAL_POOL
	}

	err = storage.DB.UpdateVMRequestStatus(ctx, storage.UpdateVMRequestStatusParams{Requestid: id, Requeststatus: storage.REQUEST_STATUS_ACCEPTED})
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	_, summary, err := proxmox.CreateVM(ctx, *opts)
	if err != nil {
		err2 := notifier.NotifyVMCreationUpdate(ctx, fmt.Sprintf("Request %d: Error creating VM:\n%v", id, "```\n"+err.Error()+"\n```"))
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

	successMsg := fmt.Sprintf("Request %v: VM %s created successfully:\n%s", request.Requestid, opts.FQDN, "```\n"+summary.String()+"\n```")
	logger.From(ctx).Info(successMsg)

	err = notifier.NotifyVMCreationUpdate(ctx, successMsg)
	if err != nil {
		return SimpleError(err, "Failed to notify VM creation update")
	}

	return nil
}

// RejectVMRequest marks a VM request as rejected.
// Returns an ErrorBundle if the request was already accepted
// or if any database/notification step fails.
func RejectVMRequest(ctx context.Context, id int64) *ErrorBundle {
	request, err := storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	if request.Requeststatus == storage.REQUEST_STATUS_ACCEPTED {
		return SimpleError(nil, "Cannot reject an accepted request")
	}

	err = storage.DB.UpdateVMRequestStatus(ctx, storage.UpdateVMRequestStatusParams{Requestid: id, Requeststatus: storage.REQUEST_STATUS_REJECTED})
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	request, err = storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	err = notifier.NotifyVMRequestStatusChanged(ctx, request, "")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	fmt.Printf("Rejected VM request %d (%s).\n", id, request.ToVMOptions().FQDN)

	return nil
}

// HoldVMRequest changes a VM request from PENDING to HELD,
// sends a status update notification, and returns any errors.
func HoldVMRequest(ctx context.Context, id int64) *ErrorBundle {
	request, err := storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	if request.Requeststatus != storage.REQUEST_STATUS_PENDING {
		return SimpleError(nil, "You can only put pending requests on hold")
	}

	err = storage.DB.UpdateVMRequestStatus(ctx, storage.UpdateVMRequestStatusParams{Requestid: id, Requeststatus: storage.REQUEST_STATUS_HELD})
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	request, err = storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	err = notifier.NotifyVMRequestStatusChanged(ctx, request, "")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	fmt.Printf("Held VM request %d (%s).\n", id, request.ToVMOptions().FQDN)

	return nil
}

// UnholdVMRequest changes a VM request from HELD to PENDING,
// sends a status update notification, and returns any errors.
func UnholdVMRequest(ctx context.Context, id int64) *ErrorBundle {
	request, err := storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	if request.Requeststatus != storage.REQUEST_STATUS_HELD {
		return SimpleError(nil, "Unhold invalid: request is not on hold")
	}

	err = storage.DB.UpdateVMRequestStatus(ctx, storage.UpdateVMRequestStatusParams{Requestid: id, Requeststatus: storage.REQUEST_STATUS_PENDING})
	if err != nil {
		return SimpleError(err, "Failed to update VM request status")
	}

	request, err = storage.DB.GetVMRequestByID(ctx, id)
	if err != nil {
		return SimpleError(err, "Failed to fetch VM request")
	}

	err = notifier.NotifyVMRequestStatusChanged(ctx, request, "")
	if err != nil {
		return SimpleError(err, "Failed to notify VM request status change")
	}

	fmt.Printf("Freed VM request %d (%s).\n", id, request.ToVMOptions().FQDN)

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

		id, err := storage.DB.CreateVMRequest(context.Background(), storage.CreateVMRequestParams{
			Email:           f.Email,
			Personalemail:   f.PersonalEmail,
			Isorganization:  f.IsOrganization,
			Orgname:         sql.NullString{String: f.OrgName, Valid: true},
			Hostname:        fmt.Sprintf("%v.vsos.ethz.ch", f.Hostname),
			Image:           f.Image,
			Cores:           int32(f.Cores),
			Ramgb:           int32(f.RamGB),
			Diskgb:          int32(f.DiskGB),
			Secondarydiskgb: int32(f.SecondaryDiskGB),
			Sshpubkeys:      f.SshPubkeys,
			Comments:        sql.NullString{String: f.Comments, Valid: true},
		})
		if err != nil {
			log.Printf("Failed to store VM request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		req, err := storage.DB.GetVMRequestByID(context.Background(), id)
		if err != nil {
			log.Printf("Failed to get VM request: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err = notifier.NotifyVMRequest(context.Background(), req)
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
		vmRequests, err := storage.DB.ListVMRequests(r.Context())
		if err != nil {
			log.Printf("Failed to get VM requests: %v", err)
			http.Error(w, "Failed to get VM requests", http.StatusInternalServerError)
			return
		}

		// API response shape, decoupled from the DB row: flatten the nullable
		// columns and keep the historical JSON field names the frontend expects.
		type vmRequestResp struct {
			ID               int64     `json:"ID"`
			RequestCreatedAt time.Time `json:"RequestCreatedAt"`
			RequestStatus    string    `json:"RequestStatus"`
			Email            string    `json:"Email"`
			PersonalEmail    string    `json:"PersonalEmail"`
			IsOrganization   bool      `json:"IsOrganization"`
			OrgName          string    `json:"OrgName"`
			Hostname         string    `json:"Hostname"`
			Image            string    `json:"Image"`
			Cores            int32     `json:"Cores"`
			RamGB            int32     `json:"RamGB"`
			DiskGB           int32     `json:"DiskGB"`
			SecondaryDiskGB  int32     `json:"SecondaryDiskGB"`
			SshPubkeys       []string  `json:"SshPubkeys"`
			Comments         string    `json:"Comments"`
		}
		out := make([]vmRequestResp, 0, len(vmRequests))
		for _, req := range vmRequests {
			out = append(out, vmRequestResp{
				ID:               req.Requestid,
				RequestCreatedAt: req.Requestcreatedat,
				RequestStatus:    string(req.Requeststatus),
				Email:            req.Email,
				PersonalEmail:    req.Personalemail,
				IsOrganization:   req.Isorganization,
				OrgName:          req.Orgname.String,
				Hostname:         req.Hostname,
				Image:            req.Image,
				Cores:            req.Cores,
				RamGB:            req.Ramgb,
				DiskGB:           req.Diskgb,
				SecondaryDiskGB:  req.Secondarydiskgb,
				SshPubkeys:       req.Sshpubkeys,
				Comments:         req.Comments.String,
			})
		}
		resp, err := json.Marshal(out)
		if err != nil {
			log.Printf("Failed to marshal VM requests: %v", err)
			http.Error(w, "Failed to marshal VM requests", http.StatusInternalServerError)
			return
		}
		w.Write(resp)
	})))

	r.Methods("POST").Path("/api/vmrequest/accept").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("accept", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		ctx, lg, finish := logger.Nest(context.Background(), fmt.Sprintf("Accept VM request %d", body.ID))
		// Set the Log Scope header such that the frontend can stream the live logs immediately.
		w.Header().Set("X-Log-Scope-Id", lg.ScopeID())
		w.WriteHeader(http.StatusAccepted)

		go func() {
			eb := AcceptVMRequest(ctx, int64(body.ID))
			if eb != nil {
				finish(eb.Err)
			} else {
				finish(nil)
			}
		}()

	}))))

	r.Methods("POST").Path("/api/vmrequest/reject").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("reject", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		eb := RejectVMRequest(r.Context(), int64(body.ID))

		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	r.Methods("POST").Path("/api/vmrequest/hold").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		eb := HoldVMRequest(r.Context(), int64(body.ID))

		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	})))

	r.Methods("POST").Path("/api/vmrequest/unhold").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		eb := UnholdVMRequest(r.Context(), int64(body.ID))

		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	})))

	r.Methods("POST").Path("/api/vmrequest/edit").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("edit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			Hostname             string `json:"hostname"`
			ID                   int    `json:"id"`
			Cores_cpu            int    `json:"cores_cpu"`
			Ram_gb               int    `json:"ram_gb"`
			Storage_gb           int    `json:"storage_gb"`
			Secondary_storage_gb int    `json:"secondary_storage_gb"`
		}

		var body bodyS
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			log.Printf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		request, err := storage.DB.GetVMRequestByID(r.Context(), int64(body.ID))
		if err != nil {
			log.Printf("Error getting VM request: %v", err)
			http.Error(w, "Failed to fetch VM request", http.StatusInternalServerError)
			return
		}

		if request.Requeststatus != storage.REQUEST_STATUS_PENDING {
			http.Error(w, "Cannot edit a request that is not pending", http.StatusBadRequest)
			return
		}

		if body.Cores_cpu != 0 {
			request.Cores = int32(body.Cores_cpu)
		}
		if body.Ram_gb != 0 {
			request.Ramgb = int32(body.Ram_gb)
		}
		if body.Storage_gb != 0 {
			request.Diskgb = int32(body.Storage_gb)
		}
		if body.Secondary_storage_gb != 0 {
			request.Secondarydiskgb = int32(body.Secondary_storage_gb)
		}
		if body.Hostname != "" {
			request.Hostname = body.Hostname
		}

		err = storage.DB.UpdateVMRequest(r.Context(), storage.UpdateVMRequestParams{
			Requestid:        request.Requestid,
			Requestcreatedat: request.Requestcreatedat,
			Requeststatus:    request.Requeststatus,
			Email:            request.Email,
			Personalemail:    request.Personalemail,
			Isorganization:   request.Isorganization,
			Orgname:          request.Orgname,
			Hostname:         request.Hostname,
			Image:            request.Image,
			Cores:            request.Cores,
			Ramgb:            request.Ramgb,
			Diskgb:           request.Diskgb,
			Secondarydiskgb:  request.Secondarydiskgb,
			Sshpubkeys:       request.Sshpubkeys,
			Comments:         request.Comments,
		})
		if err != nil {
			log.Printf("Error updating VM request: %v", err)
			http.Error(w, "Failed to update VM request", http.StatusInternalServerError)
			return
		}

	}))))
}
