package router

import (
	"context"
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
	CurrentCores           int32    `json:"CurrentCores"`
	CurrentRamGB           int32    `json:"CurrentRamGB"`
	CurrentDiskGB          int32    `json:"CurrentDiskGB"`
	CurrentSecondaryDiskGB int32    `json:"CurrentSecondaryDiskGB"`
	NewCores               int32    `json:"NewCores"`
	NewRamGB               int32    `json:"NewRamGB"`
	NewDiskGB              int32    `json:"NewDiskGB"`
	NewSecondaryDiskGB     int32    `json:"NewSecondaryDiskGB"`
	NeedsReviewReasons     []string `json:"NeedsReviewReasons"`
	ProcessedAt            *string  `json:"ProcessedAt"`
	ProcessedNote          string   `json:"ProcessedNote"`
	RequestReason          string   `json:"RequestReason"`
}

func toRcrJSON(r storage.ResourceChangeRequest) rcrJSON {
	out := rcrJSON{
		ID:                     r.ID,
		CreatedAt:              r.Createdat.Format(time.RFC3339),
		Status:                 string(r.Status),
		VMHostname:             r.Vmhostname,
		RequesterFingerprint:   r.Requesterfingerprint,
		CurrentCores:           r.Currentcores,
		CurrentRamGB:           r.Currentramgb,
		CurrentDiskGB:          r.Currentdiskgb,
		CurrentSecondaryDiskGB: r.Currentsecondarydiskgb,
		NewCores:               r.Newcores,
		NewRamGB:               r.Newramgb,
		NewDiskGB:              r.Newdiskgb,
		NewSecondaryDiskGB:     r.Newsecondarydiskgb,
		NeedsReviewReasons:     r.Needsreviewreasons,
		ProcessedNote:          r.Processednote,
		RequestReason:          r.Requestreason,
	}
	if r.Processedat.Valid {
		s := r.Processedat.Time.Format(time.RFC3339)
		out.ProcessedAt = &s
	}
	if out.NeedsReviewReasons == nil {
		out.NeedsReviewReasons = []string{}
	}
	return out
}

// evaluateHeuristic returns the list of reasons (if any) that prevent
// auto-approval. Empty list => safe to auto-approve.
func evaluateHeuristic(req storage.CreateResourceChangeParams, prev *storage.ResourceChangeRequest) []string {
	// Pure downgrade or no-op: always auto-approve, no thresholds apply.
	if req.NewCores <= req.CurrentCores &&
		req.NewRamGb <= req.CurrentRamGb &&
		req.NewDiskGb <= req.CurrentDiskGb &&
		req.NewSecondaryDiskGb <= req.CurrentSecondaryDiskGb {
		return []string{}
	}

	reasons := []string{}

	if req.NewCores > rcrThresholdCores {
		reasons = append(reasons, fmt.Sprintf("requested %d cores exceeds threshold of %d", req.NewCores, rcrThresholdCores))
	}
	if req.NewRamGb > rcrThresholdRamGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB RAM exceeds threshold of %d GB", req.NewRamGb, rcrThresholdRamGB))
	}
	if req.NewDiskGb > rcrThresholdSSDGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB SSD storage exceeds threshold of %d GB", req.NewDiskGb, rcrThresholdSSDGB))
	}
	if req.NewSecondaryDiskGb > rcrThresholdHDDGB {
		reasons = append(reasons, fmt.Sprintf("requested %d GB HDD storage exceeds threshold of %d GB", req.NewSecondaryDiskGb, rcrThresholdHDDGB))
	}
	if req.CurrentRamGb > 0 && req.NewRamGb > 2*req.CurrentRamGb {
		reasons = append(reasons, fmt.Sprintf("RAM increase from %d GB to %d GB is more than double", req.CurrentRamGb, req.NewRamGb))
	}
	if prev != nil && time.Since(prev.Createdat) < rcrRecentRequestSpan {
		reasons = append(reasons, fmt.Sprintf("previous resource change request (#%d) was less than a week ago", prev.ID))
	}

	// these heuristics probably should be improved in the future

	return reasons
}

// applyResourceChange applies the requested cores/RAM/disk change to the VM
// using the Proxmox HTTP API. Disk shrinks are rejected at submit and edit,
// so only growth is handled here.
func applyResourceChange(req storage.ResourceChangeRequest) error {
	vms, err := proxmox.GetAllClusterVMsByName(req.Vmhostname)
	if err != nil {
		return fmt.Errorf("look up VM: %v", err)
	}
	if vms == nil || len(*vms) == 0 {
		return fmt.Errorf("VM %q not found", req.Vmhostname)
	}
	if len(*vms) > 1 {
		return fmt.Errorf("VM %q is ambiguous (%d matches)", req.Vmhostname, len(*vms))
	}
	vm := (*vms)[0]

	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		return fmt.Errorf("read current disk configuration: %w", err)
	}
	if cfg.SecondaryDiskGB() != int(req.Currentsecondarydiskgb) || (req.Currentsecondarydiskgb == 0 && cfg.Scsi1 != "") {
		return fmt.Errorf("HDD configuration changed since submission; submit a new resource change request")
	}

	if err := proxmox.SetVMCoresAndMemory(vm.Node, vm.Vmid, int(req.Newcores), int(req.Newramgb)*1024); err != nil {
		return fmt.Errorf("set cores/memory: %v", err)
	}

	if req.Newdiskgb > req.Currentdiskgb {
		if err := proxmox.ResizeVMDisk(vm.Node, vm.Vmid, int(req.Currentdiskgb), int(req.Newdiskgb)); err != nil {
			return fmt.Errorf("resize disk: %v", err)
		}
	}

	// Secondary HDD: create on first request (current=0, new>0), or grow.
	// Shrinks and removals are rejected at the submit boundary, so we don't
	// need to handle them here.
	if req.Newsecondarydiskgb > 0 && req.Currentsecondarydiskgb == 0 {
		if err := proxmox.CreateVMSecondaryDisk(vm.Node, vm.Vmid, int(req.Newsecondarydiskgb)); err != nil {
			return fmt.Errorf("create secondary disk: %v", err)
		}
	} else if req.Newsecondarydiskgb > req.Currentsecondarydiskgb {
		if err := proxmox.ResizeVMSecondaryDisk(vm.Node, vm.Vmid, int(req.Currentsecondarydiskgb), int(req.Newsecondarydiskgb)); err != nil {
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
func emailUserOnApply(req storage.ResourceChangeRequest, autoApproved bool) {
	vms, err := proxmox.GetAllClusterVMsByName(req.Vmhostname)
	if err != nil || vms == nil || len(*vms) == 0 {
		log.Printf("emailUserOnApply: skipping email for %s: VM lookup failed: %v", req.Vmhostname, err)
		return
	}
	contacts := vmContacts((*vms)[0])
	if len(contacts) == 0 {
		log.Printf("emailUserOnApply: no contacts for %s, skipping email", req.Vmhostname)
		return
	}
	if err := notifier.SendResourceChangeAppliedEmail(req, contacts, autoApproved); err != nil {
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

	// Reject shrink/remove of an existing disk — Proxmox can't shrink, and
	// detaching/destroying a populated disk is destructive and needs admin care.
	if currentSecondaryDiskGB > 0 && body.NewSecondaryDiskGB < currentSecondaryDiskGB {
		return nil, &ErrorBundle{
			Err:      fmt.Errorf("HDD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink or detach an existing %d GB HDD. Ask an admin to remove it manually.", currentSecondaryDiskGB),
			HttpCode: http.StatusBadRequest,
		}
	}

	if body.NewDiskGB < currentDiskGB {
		return nil, &ErrorBundle{
			Err:      fmt.Errorf("SSD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink the existing %d GB SSD disk.", currentDiskGB),
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

	p := storage.CreateResourceChangeParams{
		Status:                 storage.RcrStatusPending,
		VmHostname:             body.VMHostname,
		RequesterFingerprint:   body.RequesterFingerprint,
		CurrentCores:           int32(currentCores),
		CurrentRamGb:           int32(currentRamGB),
		CurrentDiskGb:          int32(currentDiskGB),
		CurrentSecondaryDiskGb: int32(currentSecondaryDiskGB),
		NewCores:               int32(body.NewCores),
		NewRamGb:               int32(body.NewRamGB),
		NewDiskGb:              int32(body.NewDiskGB),
		NewSecondaryDiskGb:     int32(body.NewSecondaryDiskGB),
		RequestReason:          reason,
	}
	p.NeedsReviewReasons = evaluateHeuristic(p, prev)

	id, err := storage.DB.StoreResourceChangeRequest(p)
	if err != nil {
		return nil, SimpleError(err, "Failed to store resource change request")
	}
	rcr, err := storage.DB.GetResourceChangeByID(context.Background(), id)
	if err != nil {
		return nil, SimpleError(err, "Failed to load stored resource change request")
	}
	_ = notifier.NotifyResourceChangeRequest(rcr)

	if len(rcr.Needsreviewreasons) > 0 {
		return &rcrSubmitResponse{
			ID:                 id,
			Status:             string(storage.RcrStatusPending),
			NeedsReviewReasons: rcr.Needsreviewreasons,
			Message:            "Request stored, replacing any earlier pending request for this VM. An admin must approve it before it takes effect.",
		}, nil
	}

	// Auto-approve path.
	if err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusFailed, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(rcr, err.Error())
		return nil, SimpleError(err, "Failed to apply resource change")
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusAutoApproved, "auto-approved by heuristic"); err != nil {
		log.Printf("Failed to mark request %d as auto_approved: %v", id, err)
	}
	_ = notifier.NotifyResourceChangeRequestApplied(rcr, true)
	emailUserOnApply(rcr, true)

	return &rcrSubmitResponse{
		ID:      id,
		Status:  string(storage.RcrStatusAutoApproved),
		Message: "Request auto-approved and applied in Proxmox. Memory and CPU changes take effect on the next VM reboot",
	}, nil
}

// CancelResourceChangeRequest cancels the current pending resource change for
// the given VM. Invoked by the SSH portal on behalf of the VM owner.
func CancelResourceChangeRequest(hostname string) (*storage.ResourceChangeRequest, *ErrorBundle) {
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
	if err := storage.DB.MarkResourceChangeRequestProcessed(rcr.ID, storage.RcrStatusCancelled, "cancelled by user"); err != nil {
		return nil, SimpleError(err, "Failed to cancel request")
	}
	rcr.Status = storage.RcrStatusCancelled
	_ = notifier.NotifyResourceChangeRequestStatusChanged(*rcr)
	return rcr, nil
}

// UndoResourceChangeRequest reverts the latest applied resource change for
// the given VM by submitting a new request that restores the prior values.
// Returns the new request describing the revert.
func UndoResourceChangeRequest(hostname, fingerprint string) (*storage.ResourceChangeRequest, *ErrorBundle) {
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
	targetDiskGB := int(prev.Currentdiskgb)
	if currentDiskGB > targetDiskGB {
		targetDiskGB = currentDiskGB
	}
	targetSecondaryDiskGB := int(prev.Currentsecondarydiskgb)
	if currentSecondaryDiskGB > targetSecondaryDiskGB {
		targetSecondaryDiskGB = currentSecondaryDiskGB
	}

	p := storage.CreateResourceChangeParams{
		Status:                 storage.RcrStatusPending,
		VmHostname:             hostname,
		RequesterFingerprint:   fingerprint,
		CurrentCores:           int32(currentCores),
		CurrentRamGb:           int32(currentRamGB),
		CurrentDiskGb:          int32(currentDiskGB),
		CurrentSecondaryDiskGb: int32(currentSecondaryDiskGB),
		NewCores:               prev.Currentcores,
		NewRamGb:               prev.Currentramgb,
		NewDiskGb:              int32(targetDiskGB),
		NewSecondaryDiskGb:     int32(targetSecondaryDiskGB),
		RequestReason:          fmt.Sprintf("Undo of request #%d", prev.ID),
	}
	id, err := storage.DB.StoreResourceChangeRequest(p)
	if err != nil {
		return nil, SimpleError(err, "Failed to store undo request")
	}
	rcr, err := storage.DB.GetResourceChangeByID(context.Background(), id)
	if err != nil {
		return nil, SimpleError(err, "Failed to load stored undo request")
	}
	_ = notifier.NotifyResourceChangeRequest(rcr)

	if err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusFailed, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(rcr, err.Error())
		return nil, SimpleError(err, "Failed to apply undo")
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusAutoApproved, fmt.Sprintf("undo of request #%d", prev.ID)); err != nil {
		log.Printf("Failed to mark undo %d as auto_approved: %v", id, err)
	}
	rcr.Status = storage.RcrStatusAutoApproved
	_ = notifier.NotifyResourceChangeRequestApplied(rcr, true)
	emailUserOnApply(rcr, true)
	return &rcr, nil
}

// AcceptResourceChangeRequest is invoked from the admin console to apply a
// previously-pending resource change request.
func AcceptResourceChangeRequest(id int64) *ErrorBundle {
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()

	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be accepted", HttpCode: http.StatusBadRequest}
	}

	if err := applyResourceChange(*rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusFailed, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(*rcr, err.Error())
		return SimpleError(err, "Failed to apply resource change")
	}

	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusAccepted, "accepted by admin"); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RcrStatusAccepted
	_ = notifier.NotifyResourceChangeRequestApplied(*rcr, false)
	emailUserOnApply(*rcr, false)
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
	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be edited", HttpCode: http.StatusBadRequest}
	}
	if rcr.Currentsecondarydiskgb > 0 && secondaryDiskGB < int(rcr.Currentsecondarydiskgb) {
		return &ErrorBundle{
			Err:      fmt.Errorf("HDD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink an existing %d GB HDD via edit.", rcr.Currentsecondarydiskgb),
			HttpCode: http.StatusBadRequest,
		}
	}
	if diskGB < int(rcr.Currentdiskgb) {
		return &ErrorBundle{
			Err:      fmt.Errorf("SSD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink the %d GB SSD via edit.", rcr.Currentdiskgb),
			HttpCode: http.StatusBadRequest,
		}
	}
	err := storage.DB.UpdateResourceChangeRequestNewValues(storage.UpdateResourceChangeTargetsParams{
		ID:                 id,
		NewCores:           int32(cores),
		NewRamGb:           int32(ramGB),
		NewDiskGb:          int32(diskGB),
		NewSecondaryDiskGb: int32(secondaryDiskGB),
	})
	if err != nil {
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
	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be rejected", HttpCode: http.StatusBadRequest}
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusRejected, "rejected by admin"); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RcrStatusRejected
	_ = notifier.NotifyResourceChangeRequestStatusChanged(*rcr)
	return nil
}

// Loads the row twice: once to learn the hostname to lock, once under the lock.
func getLockedResourceChange(id int64) (*storage.ResourceChangeRequest, func(), *ErrorBundle) {
	ctx := context.Background()
	rcr, err := storage.DB.GetResourceChangeByID(ctx, id)
	if err != nil {
		return nil, nil, SimpleError(err, "Failed to fetch resource change request")
	}
	unlock, err := storage.DB.LockResourceChangeVM(rcr.Vmhostname)
	if err != nil {
		return nil, nil, SimpleError(err, "Failed to lock resource change")
	}
	rcr, err = storage.DB.GetResourceChangeByID(ctx, id)
	if err != nil {
		unlock()
		return nil, nil, SimpleError(err, "Failed to fetch resource change request")
	}
	return &rcr, unlock, nil
}

func sshportalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := config.AppConfig.SSHPORTAL_SECRET
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
		rcrs, err := storage.DB.ListResourceChanges(r.Context())
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
		_ = json.NewEncoder(w).Encode(toRcrJSON(*rcr))
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
		_ = json.NewEncoder(w).Encode(toRcrJSON(*rcr))
	})))
}
