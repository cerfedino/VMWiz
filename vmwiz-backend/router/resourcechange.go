package router

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/auth"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/config"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/confirmation"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/notifier"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/proxmox"
	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
	"github.com/gorilla/mux"
)

// Routes under /api/resourcechange/*

// Heuristic thresholds. A request that *results* in a configuration above any
// of these requires human review. The double-RAM and recent-request rules also
// apply.
const (
	rcrThresholdCores    = 8
	rcrThresholdRamGB    = 4 //currently extra low, maybe like 6/8 in the future
	rcrThresholdSSDGB    = 30
	rcrThresholdHDDGB    = 50
	rcrRecentRequestSpan = 7 * 24 * time.Hour
)

type rcrSubmitBody struct {
	VMHostname           string `json:"vmHostname"`
	RequesterFingerprint string `json:"requesterFingerprint"`
	NewCores             int    `json:"newCores"`
	NewRamGB             int    `json:"newRamGB"`
	NewDiskGB            int    `json:"newDiskGB"`
	NewSecondaryDiskGB   int    `json:"newSecondaryDiskGB"`
	Reason               string `json:"reason"`
}

type rcrCancelOrUndoBody struct {
	VMHostname           string `json:"vmHostname"`
	RequesterFingerprint string `json:"requesterFingerprint"`
}

type rcrSubmitResponse struct {
	ID                 int64    `json:"id"`
	Status             string   `json:"status"`
	NeedsReviewReasons []string `json:"needsReviewReasons"`
	Message            string   `json:"message"`
}

// rcrJSON is the JSON shape returned by the admin list endpoint. It mirrors
// the SQL row but flattens sql.NullTime into a plain string-or-null.
type rcrJSON struct {
	ID                     int64    `json:"ID"`
	CreatedAt              string   `json:"CreatedAt"`
	Status                 string   `json:"Status"`
	VMHostname             string   `json:"VMHostname"`
	RequesterFingerprint   string   `json:"RequesterFingerprint"`
	CurrentCores           int      `json:"CurrentCores"`
	CurrentRamGB           int      `json:"CurrentRamGB"`
	CurrentDiskGB          int      `json:"CurrentDiskGB"`
	CurrentSecondaryDiskGB int      `json:"CurrentSecondaryDiskGB"`
	NewCores               int      `json:"NewCores"`
	NewRamGB               int      `json:"NewRamGB"`
	NewDiskGB              int      `json:"NewDiskGB"`
	NewSecondaryDiskGB     int      `json:"NewSecondaryDiskGB"`
	NeedsReviewReasons     []string `json:"NeedsReviewReasons"`
	ProcessedAt            *string  `json:"ProcessedAt"`
	ProcessedNote          string   `json:"ProcessedNote"`
	RequestReason          string   `json:"RequestReason"`
}

func toRcrJSON(r *storage.SQLResourceChangeRequest) rcrJSON {
	out := rcrJSON{
		ID:                     r.ID,
		CreatedAt:              r.CreatedAt.Format(time.RFC3339),
		Status:                 r.Status,
		VMHostname:             r.VMHostname,
		RequesterFingerprint:   r.RequesterFingerprint,
		CurrentCores:           r.CurrentCores,
		CurrentRamGB:           r.CurrentRamGB,
		CurrentDiskGB:          r.CurrentDiskGB,
		CurrentSecondaryDiskGB: r.CurrentSecondaryDiskGB,
		NewCores:               r.NewCores,
		NewRamGB:               r.NewRamGB,
		NewDiskGB:              r.NewDiskGB,
		NewSecondaryDiskGB:     r.NewSecondaryDiskGB,
		NeedsReviewReasons:     r.NeedsReviewReasons,
		ProcessedNote:          r.ProcessedNote,
		RequestReason:          r.RequestReason,
	}
	if r.ProcessedAt.Valid {
		s := r.ProcessedAt.Time.Format(time.RFC3339)
		out.ProcessedAt = &s
	}
	if out.NeedsReviewReasons == nil {
		out.NeedsReviewReasons = []string{}
	}
	return out
}

// evaluateHeuristic returns the list of reasons (if any) that prevent
// auto-approval. Empty list => safe to auto-approve.
func evaluateHeuristic(req *storage.SQLResourceChangeRequest, prev *storage.SQLResourceChangeRequest) []string {
	// Pure downgrade or no-op: always auto-approve, no thresholds apply.
	if req.NewCores <= req.CurrentCores &&
		req.NewRamGB <= req.CurrentRamGB &&
		req.NewDiskGB <= req.CurrentDiskGB &&
		req.NewSecondaryDiskGB <= req.CurrentSecondaryDiskGB {
		return []string{}
	}

	reasons := []string{}

	if req.NewCores > rcrThresholdCores {
		reasons = append(reasons, fmt.Sprintf("requested %d cores exceeds threshold of %d", req.NewCores, rcrThresholdCores))
	}
	if req.NewRamGB > rcrThresholdRamGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB RAM exceeds threshold of %d GB", req.NewRamGB, rcrThresholdRamGB))
	}
	if req.NewDiskGB > rcrThresholdSSDGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB SSD storage exceeds threshold of %d GB", req.NewDiskGB, rcrThresholdSSDGB))
	}
	if req.NewSecondaryDiskGB > rcrThresholdHDDGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB HDD storage exceeds threshold of %d GB", req.NewSecondaryDiskGB, rcrThresholdHDDGB))
	}
	if req.CurrentRamGB > 0 && req.NewRamGB > 2*req.CurrentRamGB {
		reasons = append(reasons, fmt.Sprintf("RAM increase from %d GB to %d GB is more than double", req.CurrentRamGB, req.NewRamGB))
	}
	if prev != nil && time.Since(prev.CreatedAt) < rcrRecentRequestSpan {
		reasons = append(reasons, fmt.Sprintf("previous resource change request (#%d) was less than a week ago", prev.ID))
	}

	// these heuristics probably should be improved in the future

	return reasons
}

// applyResourceChange applies the requested cores/RAM/disk change to the VM
// using the Proxmox HTTP API. Disk shrinks are not supported by Proxmox and
// will be left unchanged with a warning in the returned message.
func applyResourceChange(req *storage.SQLResourceChangeRequest) error {
	vms, err := proxmox.GetAllClusterVMsByName(req.VMHostname)
	if err != nil {
		return fmt.Errorf("look up VM: %v", err)
	}
	if vms == nil || len(*vms) == 0 {
		return fmt.Errorf("VM %q not found", req.VMHostname)
	}
	if len(*vms) > 1 {
		return fmt.Errorf("VM %q is ambiguous (%d matches)", req.VMHostname, len(*vms))
	}
	vm := (*vms)[0]

	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		return fmt.Errorf("read current disk configuration: %w", err)
	}
	if cfg.SecondaryDiskGB() != req.CurrentSecondaryDiskGB || (req.CurrentSecondaryDiskGB == 0 && cfg.Scsi1 != "") {
		return fmt.Errorf("HDD configuration changed since submission; submit a new resource change request")
	}

	if err := proxmox.SetVMCoresAndMemory(vm.Node, vm.Vmid, req.NewCores, req.NewRamGB*1024); err != nil {
		return fmt.Errorf("set cores/memory: %v", err)
	}

	if req.NewDiskGB > req.CurrentDiskGB {
		if err := proxmox.ResizeVMDisk(vm.Node, vm.Vmid, req.CurrentDiskGB, req.NewDiskGB); err != nil {
			return fmt.Errorf("resize disk: %v", err)
		}
	}

	// Secondary HDD: create on first request (current=0, new>0), or grow.
	// Shrinks and removals are rejected at the submit boundary, so we don't
	// need to handle them here.
	if req.NewSecondaryDiskGB > 0 && req.CurrentSecondaryDiskGB == 0 {
		if err := proxmox.CreateVMSecondaryDisk(vm.Node, vm.Vmid, req.NewSecondaryDiskGB); err != nil {
			return fmt.Errorf("create secondary disk: %v", err)
		}
	} else if req.NewSecondaryDiskGB > req.CurrentSecondaryDiskGB {
		if err := proxmox.ResizeVMSecondaryDisk(vm.Node, vm.Vmid, req.CurrentSecondaryDiskGB, req.NewSecondaryDiskGB); err != nil {
			return fmt.Errorf("resize secondary disk: %v", err)
		}
	}

	return nil
}

// vmContacts returns the contact email addresses for a VM by reading its
// Proxmox config description. Failures are logged and an empty slice returned
// email notification is best-effort.
func vmContacts(vm proxmox.PVEClusterVM) []string {
	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		log.Printf("vmContacts: failed to fetch VM config for %s: %v", vm.Name, err)
		return nil
	}
	return proxmox.GetEmails(*cfg, nil)
}

// emailUserOnApply looks up the VM, extracts contact emails, and sends the
// "resource change applied" email. Failures are logged and ignored.
func emailUserOnApply(req *storage.SQLResourceChangeRequest, autoApproved bool) {
	vms, err := proxmox.GetAllClusterVMsByName(req.VMHostname)
	if err != nil || vms == nil || len(*vms) == 0 {
		log.Printf("emailUserOnApply: skipping email for %s: VM lookup failed: %v", req.VMHostname, err)
		return
	}
	contacts := vmContacts((*vms)[0])
	if len(contacts) == 0 {
		log.Printf("emailUserOnApply: no contacts for %s, skipping email", req.VMHostname)
		return
	}
	if err := notifier.SendResourceChangeAppliedEmail(*req, contacts, autoApproved); err != nil {
		log.Printf("emailUserOnApply: send email for request %d failed: %v", req.ID, err)
	}
}

// SubmitResourceChangeRequest handles a request from the SSH portal. It
// fetches the current VM resources, runs the heuristic, stores the request,
// and either auto-applies or marks it pending for human review.
func SubmitResourceChangeRequest(body *rcrSubmitBody) (*rcrSubmitResponse, *ErrorBundle) {
	if body.VMHostname == "" {
		return nil, &ErrorBundle{Err: fmt.Errorf("missing vmHostname"), UserMsg: "missing vmHostname", HttpCode: http.StatusBadRequest}
	}
	if body.NewCores <= 0 || body.NewRamGB <= 0 || body.NewDiskGB <= 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("invalid resource values"), UserMsg: "cores, ramGB, diskGB must all be positive", HttpCode: http.StatusBadRequest}
	}
	if body.NewSecondaryDiskGB < 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("invalid secondary disk"), UserMsg: "secondaryDiskGB must be >= 0", HttpCode: http.StatusBadRequest}
	}

	unlock, err := storage.DB.LockResourceChangeVM(body.VMHostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to lock resource change")
	}
	defer unlock()

	vms, err := proxmox.GetAllClusterVMsByName(body.VMHostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to look up VM")
	}
	if vms == nil || len(*vms) == 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("VM not found"), UserMsg: "VM not found", HttpCode: http.StatusNotFound}
	}
	if len(*vms) > 1 {
		return nil, &ErrorBundle{Err: fmt.Errorf("VM ambiguous"), UserMsg: "VM hostname is ambiguous", HttpCode: http.StatusConflict}
	}
	vm := (*vms)[0]

	currentCores := int(vm.Maxcpu)
	currentRamGB := vm.Maxmem / (1024 * 1024 * 1024)
	currentDiskGB := vm.Maxdisk / (1024 * 1024 * 1024)

	// Require a readable VM config; a failed lookup must not imply an empty HDD slot.
	currentSecondaryDiskGB := 0
	if cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid); err == nil {
		currentSecondaryDiskGB = cfg.SecondaryDiskGB()
	} else {
		return nil, SimpleError(err, "Failed to read current VM disk configuration")
	}

	// Reject shrink/remove of an existing HDD — Proxmox can't shrink, and
	// detaching/destroying a populated HDD is destructive and needs admin care.
	if currentSecondaryDiskGB > 0 && body.NewSecondaryDiskGB < currentSecondaryDiskGB {
		return nil, &ErrorBundle{
			Err:      fmt.Errorf("HDD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink or detach an existing %d GB HDD. Ask an admin to remove it manually.", currentSecondaryDiskGB),
			HttpCode: http.StatusBadRequest,
		}
	}

	// Require a justification for any upward change.
	isIncrease := body.NewCores > currentCores ||
		body.NewRamGB > currentRamGB ||
		body.NewDiskGB > currentDiskGB ||
		body.NewSecondaryDiskGB > currentSecondaryDiskGB
	reason := strings.TrimSpace(body.Reason)
	if isIncrease && reason == "" {
		return nil, &ErrorBundle{Err: fmt.Errorf("missing reason"), UserMsg: "A reason is required when increasing any resource.", HttpCode: http.StatusBadRequest}
	}

	prev, err := storage.DB.GetLatestProcessedResourceChangeRequestForHostname(body.VMHostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to look up previous requests")
	}

	rcr := &storage.SQLResourceChangeRequest{
		Status:                 storage.RCR_STATUS_PENDING,
		VMHostname:             body.VMHostname,
		RequesterFingerprint:   body.RequesterFingerprint,
		CurrentCores:           currentCores,
		CurrentRamGB:           currentRamGB,
		CurrentDiskGB:          currentDiskGB,
		CurrentSecondaryDiskGB: currentSecondaryDiskGB,
		NewCores:               body.NewCores,
		NewRamGB:               body.NewRamGB,
		NewDiskGB:              body.NewDiskGB,
		NewSecondaryDiskGB:     body.NewSecondaryDiskGB,
		RequestReason:          reason,
	}
	rcr.NeedsReviewReasons = evaluateHeuristic(rcr, prev)

	id, err := storage.DB.StoreResourceChangeRequest(rcr)
	if err != nil {
		return nil, SimpleError(err, "Failed to store resource change request")
	}
	rcr.ID = id
	_ = notifier.NotifyResourceChangeRequest(*rcr)

	if len(rcr.NeedsReviewReasons) > 0 {
		return &rcrSubmitResponse{
			ID:                 id,
			Status:             storage.RCR_STATUS_PENDING,
			NeedsReviewReasons: rcr.NeedsReviewReasons,
			Message:            "Request stored, replacing any earlier pending request for this VM. An admin must approve it before it takes effect.",
		}, nil
	}

	// Auto-approve path.
	if err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_FAILED, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(*rcr, err.Error())
		return nil, SimpleError(err, "Failed to apply resource change")
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_AUTO_APPROVED, "auto-approved by heuristic"); err != nil {
		log.Printf("Failed to mark request %d as auto_approved: %v", id, err)
	}
	_ = notifier.NotifyResourceChangeRequestApplied(*rcr, true)
	emailUserOnApply(rcr, true)

	return &rcrSubmitResponse{
		ID:      id,
		Status:  storage.RCR_STATUS_AUTO_APPROVED,
		Message: "Request auto-approved and applied in Proxmox. Memory and CPU changes take effect on the next VM reboot",
	}, nil
}

// CancelResourceChangeRequest cancels the current pending resource change for
// the given VM. Invoked by the SSH portal on behalf of the VM owner.
func CancelResourceChangeRequest(hostname string) (*storage.SQLResourceChangeRequest, *ErrorBundle) {
	if hostname == "" {
		return nil, &ErrorBundle{Err: fmt.Errorf("missing vmHostname"), UserMsg: "missing vmHostname", HttpCode: http.StatusBadRequest}
	}
	unlock, err := storage.DB.LockResourceChangeVM(hostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to lock resource change")
	}
	defer unlock()

	rcr, err := storage.DB.GetLatestPendingResourceChangeRequestForHostname(hostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to look up pending request")
	}
	if rcr == nil {
		return nil, &ErrorBundle{Err: fmt.Errorf("no pending request"), UserMsg: "No pending resource change request to cancel.", HttpCode: http.StatusNotFound}
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(rcr.ID, storage.RCR_STATUS_CANCELLED, "cancelled by user"); err != nil {
		return nil, SimpleError(err, "Failed to cancel request")
	}
	rcr.Status = storage.RCR_STATUS_CANCELLED
	_ = notifier.NotifyResourceChangeRequestStatusChanged(*rcr)
	return rcr, nil
}

// UndoResourceChangeRequest reverts the latest applied resource change for
// the given VM by submitting a new request that restores the prior values.
// Returns the new request describing the revert.
func UndoResourceChangeRequest(hostname, fingerprint string) (*storage.SQLResourceChangeRequest, *ErrorBundle) {
	if hostname == "" {
		return nil, &ErrorBundle{Err: fmt.Errorf("missing vmHostname"), UserMsg: "missing vmHostname", HttpCode: http.StatusBadRequest}
	}
	unlock, err := storage.DB.LockResourceChangeVM(hostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to lock resource change")
	}
	defer unlock()

	prev, err := storage.DB.GetLatestProcessedResourceChangeRequestForHostname(hostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to look up previous applied request")
	}
	if prev == nil {
		return nil, &ErrorBundle{Err: fmt.Errorf("nothing to undo"), UserMsg: "No applied resource change to undo.", HttpCode: http.StatusNotFound}
	}

	vms, err := proxmox.GetAllClusterVMsByName(hostname)
	if err != nil {
		return nil, SimpleError(err, "Failed to look up VM")
	}
	if vms == nil || len(*vms) == 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("VM not found"), UserMsg: "VM not found", HttpCode: http.StatusNotFound}
	}
	vm := (*vms)[0]

	currentCores := int(vm.Maxcpu)
	currentRamGB := vm.Maxmem / (1024 * 1024 * 1024)
	currentDiskGB := vm.Maxdisk / (1024 * 1024 * 1024)
	currentSecondaryDiskGB := 0
	if cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid); err == nil {
		currentSecondaryDiskGB = cfg.SecondaryDiskGB()
	}

	// Disk shrinks aren't supported by Proxmox; clamp the target disk to the
	// larger of (prior current disk, current disk).
	targetDiskGB := prev.CurrentDiskGB
	if currentDiskGB > targetDiskGB {
		targetDiskGB = currentDiskGB
	}
	targetSecondaryDiskGB := prev.CurrentSecondaryDiskGB
	if currentSecondaryDiskGB > targetSecondaryDiskGB {
		targetSecondaryDiskGB = currentSecondaryDiskGB
	}

	rcr := &storage.SQLResourceChangeRequest{
		Status:                 storage.RCR_STATUS_PENDING,
		VMHostname:             hostname,
		RequesterFingerprint:   fingerprint,
		CurrentCores:           currentCores,
		CurrentRamGB:           currentRamGB,
		CurrentDiskGB:          currentDiskGB,
		CurrentSecondaryDiskGB: currentSecondaryDiskGB,
		NewCores:               prev.CurrentCores,
		NewRamGB:               prev.CurrentRamGB,
		NewDiskGB:              targetDiskGB,
		NewSecondaryDiskGB:     targetSecondaryDiskGB,
		RequestReason:          fmt.Sprintf("Undo of request #%d", prev.ID),
	}
	id, err := storage.DB.StoreResourceChangeRequest(rcr)
	if err != nil {
		return nil, SimpleError(err, "Failed to store undo request")
	}
	rcr.ID = id
	_ = notifier.NotifyResourceChangeRequest(*rcr)

	if err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_FAILED, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(*rcr, err.Error())
		return nil, SimpleError(err, "Failed to apply undo")
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_AUTO_APPROVED, fmt.Sprintf("undo of request #%d", prev.ID)); err != nil {
		log.Printf("Failed to mark undo %d as auto_approved: %v", id, err)
	}
	rcr.Status = storage.RCR_STATUS_AUTO_APPROVED
	_ = notifier.NotifyResourceChangeRequestApplied(*rcr, true)
	emailUserOnApply(rcr, true)
	return rcr, nil
}

// AcceptResourceChangeRequest is invoked from the admin console to apply a
// previously-pending resource change request.
func AcceptResourceChangeRequest(id int64) *ErrorBundle {
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()

	if rcr.Status != storage.RCR_STATUS_PENDING {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be accepted", HttpCode: http.StatusBadRequest}
	}

	if err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_FAILED, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(*rcr, err.Error())
		return SimpleError(err, "Failed to apply resource change")
	}

	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_ACCEPTED, "accepted by admin"); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RCR_STATUS_ACCEPTED
	_ = notifier.NotifyResourceChangeRequestApplied(*rcr, false)
	emailUserOnApply(rcr, false)
	return nil
}

// EditResourceChangeRequest overwrites the requested target values on a
// pending request. The edited request remains pending for admin approval.
// Returns an error if the request is not pending.
func EditResourceChangeRequest(id int64, cores, ramGB, diskGB, secondaryDiskGB int) *ErrorBundle {
	if cores <= 0 || ramGB <= 0 || diskGB <= 0 {
		return &ErrorBundle{Err: fmt.Errorf("invalid"), UserMsg: "cores, ramGB, diskGB must all be positive", HttpCode: http.StatusBadRequest}
	}
	if secondaryDiskGB < 0 {
		return &ErrorBundle{Err: fmt.Errorf("invalid"), UserMsg: "secondaryDiskGB must be >= 0", HttpCode: http.StatusBadRequest}
	}
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()
	if rcr.Status != storage.RCR_STATUS_PENDING {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be edited", HttpCode: http.StatusBadRequest}
	}
	if rcr.CurrentSecondaryDiskGB > 0 && secondaryDiskGB < rcr.CurrentSecondaryDiskGB {
		return &ErrorBundle{
			Err:      fmt.Errorf("HDD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink an existing %d GB HDD via edit.", rcr.CurrentSecondaryDiskGB),
			HttpCode: http.StatusBadRequest,
		}
	}
	if err := storage.DB.UpdateResourceChangeRequestNewValues(id, cores, ramGB, diskGB, secondaryDiskGB); err != nil {
		return SimpleError(err, "Failed to update resource change request")
	}
	return nil
}

func RejectResourceChangeRequest(id int64) *ErrorBundle {
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()
	if rcr.Status != storage.RCR_STATUS_PENDING {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be rejected", HttpCode: http.StatusBadRequest}
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RCR_STATUS_REJECTED, "rejected by admin"); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RCR_STATUS_REJECTED
	_ = notifier.NotifyResourceChangeRequestStatusChanged(*rcr)
	return nil
}

func getLockedResourceChange(id int64) (*storage.SQLResourceChangeRequest, func(), *ErrorBundle) {
	rcr, err := storage.DB.GetResourceChangeRequestById(id)
	if err != nil {
		return nil, nil, SimpleError(err, "Failed to fetch resource change request")
	}
	unlock, err := storage.DB.LockResourceChangeVM(rcr.VMHostname)
	if err != nil {
		return nil, nil, SimpleError(err, "Failed to lock resource change")
	}
	rcr, err = storage.DB.GetResourceChangeRequestById(id)
	if err != nil {
		unlock()
		return nil, nil, SimpleError(err, "Failed to fetch resource change request")
	}
	return rcr, unlock, nil
}

func sshportalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := config.AppConfig.SSHPORTAL_API_TOKEN
		if token == "" {
			http.Error(w, "ssh portal integration not configured", http.StatusServiceUnavailable)
			return
		}
		hdr := r.Header.Get("Authorization")
		if hdr != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func addAllResourceChangeRoutes(r *mux.Router) {

	// Submit endpoint, called by the SSH portal on behalf of an
	// authenticated VM owner. Authenticated via shared bearer token.
	r.Methods("POST").Path("/api/resourcechange").Handler(sshportalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rcrSubmitBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Printf("Error decoding resource change body: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		resp, eb := SubmitResourceChangeRequest(&body)
		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})))

	// Admin: list all
	r.Methods("GET").Path("/api/resourcechange").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rcrs, err := storage.DB.GetAllResourceChangeRequests()
		if err != nil {
			log.Printf("Failed to get resource change requests: %v", err)
			http.Error(w, "Failed to get resource change requests", http.StatusInternalServerError)
			return
		}
		out := make([]rcrJSON, 0, len(rcrs))
		for _, rcr := range rcrs {
			out = append(out, toRcrJSON(rcr))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})))

	// Admin: accept (confirmable — user must type "change spec" to confirm)
	r.Methods("POST").Path("/api/resourcechange/accept").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("change spec", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID int `json:"id"`
		}
		var b bodyS
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if eb := AcceptResourceChangeRequest(int64(b.ID)); eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	// Admin: reject (confirmable — user must type "reject" to confirm)
	r.Methods("POST").Path("/api/resourcechange/reject").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("reject", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID int `json:"id"`
		}
		var b bodyS
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if eb := RejectResourceChangeRequest(int64(b.ID)); eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	// Admin: edit pending request (confirmable — type "edit")
	r.Methods("POST").Path("/api/resourcechange/edit").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("edit", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID                 int `json:"id"`
			NewCores           int `json:"newCores"`
			NewRamGB           int `json:"newRamGB"`
			NewDiskGB          int `json:"newDiskGB"`
			NewSecondaryDiskGB int `json:"newSecondaryDiskGB"`
		}
		var b bodyS
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if eb := EditResourceChangeRequest(int64(b.ID), b.NewCores, b.NewRamGB, b.NewDiskGB, b.NewSecondaryDiskGB); eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	// SSH portal: cancel pending request for a VM
	r.Methods("POST").Path("/api/resourcechange/cancel").Handler(sshportalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rcrCancelOrUndoBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		rcr, eb := CancelResourceChangeRequest(body.VMHostname)
		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toRcrJSON(rcr))
	})))

	// SSH portal: undo last applied change for a VM
	r.Methods("POST").Path("/api/resourcechange/undo").Handler(sshportalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rcrCancelOrUndoBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		rcr, eb := UndoResourceChangeRequest(body.VMHostname, body.RequesterFingerprint)
		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toRcrJSON(rcr))
	})))
}
