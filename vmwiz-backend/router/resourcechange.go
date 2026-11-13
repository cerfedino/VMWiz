package router

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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

type rcrCancelBody struct {
	VMHostname string `json:"vmHostname"`
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
		CreatedAt:              r.CreatedAt.Format(time.RFC3339),
		Status:                 string(r.Status),
		VMHostname:             r.VmHostname,
		RequesterFingerprint:   r.RequesterFingerprint,
		CurrentCores:           r.CurrentCores,
		CurrentRamGB:           r.CurrentRamGb,
		CurrentDiskGB:          r.CurrentDiskGb,
		CurrentSecondaryDiskGB: r.CurrentSecondaryDiskGb,
		NewCores:               r.NewCores,
		NewRamGB:               r.NewRamGb,
		NewDiskGB:              r.NewDiskGb,
		NewSecondaryDiskGB:     r.NewSecondaryDiskGb,
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
	if prev != nil && time.Since(prev.CreatedAt) < rcrRecentRequestSpan {
		reasons = append(reasons, fmt.Sprintf("previous resource change request (#%d) was less than a week ago", prev.ID))
	}

	// these heuristics probably should be improved in the future

	return reasons
}

// applyResourceChange applies the requested cores/RAM/disk change to the VM
// using the Proxmox HTTP API. Disk shrinks are rejected at submit and edit,
// so only growth is handled here.
func applyResourceChange(req storage.ResourceChangeRequest) (partial bool, err error) {
	var applied []string
	fail := func(step string, err error) (bool, error) {
		if len(applied) == 0 {
			return false, fmt.Errorf("%s: %v", step, err)
		}
		return true, fmt.Errorf("%s: %v (already applied: %s)", step, err, strings.Join(applied, ", "))
	}

	vms, err := proxmox.GetAllClusterVMsByName(req.VmHostname)
	if err != nil {
		return fail("look up VM", err)
	}
	if vms == nil || len(*vms) == 0 {
		return false, fmt.Errorf("VM %q not found", req.VmHostname)
	}
	if len(*vms) > 1 {
		return false, fmt.Errorf("VM %q is ambiguous (%d matches)", req.VmHostname, len(*vms))
	}
	vm := (*vms)[0]

	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		return fail("read current disk configuration", err)
	}
	hddDone := req.NewSecondaryDiskGb > 0 && cfg.SecondaryDiskGB() == int(req.NewSecondaryDiskGb)
	if !hddDone && (cfg.SecondaryDiskGB() != int(req.CurrentSecondaryDiskGb) || (req.CurrentSecondaryDiskGb == 0 && cfg.Scsi3 != "")) {
		return false, fmt.Errorf("HDD configuration changed since submission; submit a new resource change request")
	}

	if req.NewDiskGb > req.CurrentDiskGb {
		if err := proxmox.ResizeVMDisk(vm.Node, vm.Vmid, int(req.CurrentDiskGb), int(req.NewDiskGb)); err != nil {
			return fail("resize disk", err)
		}
		applied = append(applied, fmt.Sprintf("SSD %d -> %d GB", req.CurrentDiskGb, req.NewDiskGb))
	}

	// Secondary HDD: create on first request (current=0, new>0), or grow.
	// Shrinks and removals are rejected at the submit boundary, so we don't
	// need to handle them here.
	switch {
	case hddDone:
	case req.NewSecondaryDiskGb > 0 && req.CurrentSecondaryDiskGb == 0:
		if err := proxmox.CreateVMSecondaryDisk(vm.Node, vm.Vmid, int(req.NewSecondaryDiskGb)); err != nil {
			return fail("create secondary disk", err)
		}
		applied = append(applied, fmt.Sprintf("HDD 0 -> %d GB", req.NewSecondaryDiskGb))
	case req.NewSecondaryDiskGb > req.CurrentSecondaryDiskGb:
		if err := proxmox.ResizeVMSecondaryDisk(vm.Node, vm.Vmid, int(req.CurrentSecondaryDiskGb), int(req.NewSecondaryDiskGb)); err != nil {
			return fail("resize secondary disk", err)
		}
		applied = append(applied, fmt.Sprintf("HDD %d -> %d GB", req.CurrentSecondaryDiskGb, req.NewSecondaryDiskGb))
	}

	cores, memoryMB := 0, 0
	if req.NewCores != req.CurrentCores {
		cores = int(req.NewCores)
	}
	if req.NewRamGb != req.CurrentRamGb {
		memoryMB = int(req.NewRamGb) * 1024
	}
	if err := proxmox.SetVMCoresAndMemory(vm.Node, vm.Vmid, cores, memoryMB); err != nil {
		return fail("set cores/memory", err)
	}

	return false, nil
}

// vmOwnerContacts returns the contact addresses from the VM's Proxmox
// description, or SMTP_RECEIVER_OVERRIDE when set. Owner email is best-effort:
// failures are logged and give nil.
func vmOwnerContacts(hostname string) []string {
	vms, err := proxmox.GetAllClusterVMsByName(hostname)
	if err != nil || vms == nil || len(*vms) == 0 {
		log.Printf("vmOwnerContacts: VM lookup for %s failed: %v", hostname, err)
		return nil
	}
	vm := (*vms)[0]
	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		log.Printf("vmOwnerContacts: failed to fetch VM config for %s: %v", hostname, err)
		return nil
	}
	contacts := proxmox.GetEmails(*cfg, nil)
	if len(contacts) == 0 {
		log.Printf("vmOwnerContacts: no contacts for %s", hostname)
		return nil
	}
	if config.AppConfig.SMTP_RECEIVER_OVERRIDE != "" {
		return []string{config.AppConfig.SMTP_RECEIVER_OVERRIDE}
	}
	return contacts
}

func emailUserOnApply(req storage.ResourceChangeRequest, autoApproved bool, note string) {
	if err := notifier.SendResourceChangeAppliedEmail(req, vmOwnerContacts(req.VmHostname), autoApproved, note); err != nil {
		log.Printf("emailUserOnApply: send email for request %d failed: %v", req.ID, err)
	}
}

// SubmitResourceChangeRequest handles a request from the SSH portal. It
// fetches the current VM resources, runs the heuristic, stores the request,
// and either auto-applies or marks it pending for human review.
func SubmitResourceChangeRequest(ctx context.Context, body *rcrSubmitBody) (*rcrSubmitResponse, *ErrorBundle) {
	closure, err := storage.DB.CurrentResourceChangeClosure(ctx)
	if err != nil {
		return nil, SimpleError(err, "Failed to get resource change closure")
	}
	if closure != nil {
		return nil, &ErrorBundle{Err: fmt.Errorf("resource changes closed"), UserMsg: "Resource change requests are currently closed: " + closure.Reason, HttpCode: http.StatusServiceUnavailable}
	}
	if body.VMHostname == "" {
		return nil, &ErrorBundle{Err: fmt.Errorf("missing vmHostname"), UserMsg: "missing vmHostname", HttpCode: http.StatusBadRequest}
	}
	if body.NewCores <= 0 || body.NewRamGB <= 0 || body.NewDiskGB <= 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("invalid resource values"), UserMsg: "cores, ramGB, diskGB must all be positive", HttpCode: http.StatusBadRequest}
	}
	if body.NewSecondaryDiskGB < 0 {
		return nil, &ErrorBundle{Err: fmt.Errorf("invalid secondary disk"), UserMsg: "secondaryDiskGB must be >= 0", HttpCode: http.StatusBadRequest}
	}

	if max(body.NewCores, body.NewRamGB, body.NewDiskGB, body.NewSecondaryDiskGB) > math.MaxInt32 {
		return nil, &ErrorBundle{Err: fmt.Errorf("resource value too large"), UserMsg: "cores, ramGB, diskGB, secondaryDiskGB are too large", HttpCode: http.StatusBadRequest}
	}
	reason := strings.TrimSpace(body.Reason)
	if utf8.RuneCountInString(reason) > 1000 {
		return nil, &ErrorBundle{Err: fmt.Errorf("reason too long"), UserMsg: "The reason must be at most 1000 characters.", HttpCode: http.StatusBadRequest}
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

	cfg, err := proxmox.GetNodeVMConfig(vm.Node, vm.Vmid)
	if err != nil {
		return nil, SimpleError(err, "Failed to read current VM configuration")
	}
	currentCores := cfg.Cores
	currentRamGB := cfg.MemoryMB() / 1024
	currentDiskGB := vm.Maxdisk / (1024 * 1024 * 1024)
	currentSecondaryDiskGB := cfg.SecondaryDiskGB()

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
	_ = notifier.NotifyResourceChangeRequest(ctx, rcr)

	if len(rcr.NeedsReviewReasons) > 0 {
		return &rcrSubmitResponse{
			ID:                 id,
			Status:             string(storage.RcrStatusPending),
			NeedsReviewReasons: rcr.NeedsReviewReasons,
			Message:            "Request stored, replacing any earlier pending request for this VM. An admin must approve it before it takes effect.",
		}, nil
	}

	// Auto-approve path.
	if _, err := applyResourceChange(rcr); err != nil {
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusFailed, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(ctx, rcr, err.Error())
		return nil, SimpleError(err, "Failed to apply resource change")
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusAutoApproved, "auto-approved by heuristic"); err != nil {
		log.Printf("Failed to mark request %d as auto_approved: %v", id, err)
	}
	_ = notifier.NotifyResourceChangeRequestApplied(ctx, rcr, true)
	emailUserOnApply(rcr, true, "")

	return &rcrSubmitResponse{
		ID:      id,
		Status:  string(storage.RcrStatusAutoApproved),
		Message: "Request auto-approved and applied in Proxmox. Memory and CPU changes take effect on the next VM reboot",
	}, nil
}

// CancelResourceChangeRequest cancels the current pending resource change for
// the given VM. Invoked by the SSH portal on behalf of the VM owner.
func CancelResourceChangeRequest(ctx context.Context, hostname string) (*storage.ResourceChangeRequest, *ErrorBundle) {
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
	_ = notifier.NotifyResourceChangeRequestStatusChanged(ctx, *rcr)
	return rcr, nil
}

// AcceptResourceChangeRequest is invoked from the admin console to apply a
// previously-pending resource change request.
func AcceptResourceChangeRequest(ctx context.Context, id int64, reason string) *ErrorBundle {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > 1000 {
		return &ErrorBundle{Err: fmt.Errorf("reason too long"), UserMsg: "The reason must be at most 1000 characters.", HttpCode: http.StatusBadRequest}
	}
	note := reason
	if note == "" {
		note = "accepted by admin"
	}
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()

	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be accepted", HttpCode: http.StatusBadRequest}
	}

	if partial, err := applyResourceChange(*rcr); err != nil {
		log.Printf("Failed to apply resource change request %d: %v", id, err)
		// Stay pending so the admin can retry, edit or reject. The failing step itself may still have taken effect.
		if !partial {
			return &ErrorBundle{Err: err, UserMsg: fmt.Sprintf("No step completed, the request stays pending: %v", err), HttpCode: http.StatusInternalServerError}
		}
		_ = storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusFailed, err.Error())
		_ = notifier.NotifyResourceChangeRequestFailed(ctx, *rcr, err.Error())
		return &ErrorBundle{Err: err, UserMsg: fmt.Sprintf("Resource change partly applied: %v", err), HttpCode: http.StatusInternalServerError}
	}

	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusAccepted, note); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RcrStatusAccepted
	_ = notifier.NotifyResourceChangeRequestApplied(ctx, *rcr, false)
	emailUserOnApply(*rcr, false, reason)
	return nil
}

// EditResourceChangeRequest overwrites the requested target values on a
// pending request. The edited request remains pending for admin approval.
// Returns an error if the request is not pending.
func EditResourceChangeRequest(ctx context.Context, id int64, cores, ramGB, diskGB, secondaryDiskGB int) *ErrorBundle {
	if cores <= 0 || ramGB <= 0 || diskGB <= 0 {
		return &ErrorBundle{Err: fmt.Errorf("invalid"), UserMsg: "cores, ramGB, diskGB must all be positive", HttpCode: http.StatusBadRequest}
	}
	if secondaryDiskGB < 0 {
		return &ErrorBundle{Err: fmt.Errorf("invalid"), UserMsg: "secondaryDiskGB must be >= 0", HttpCode: http.StatusBadRequest}
	}

	if max(cores, ramGB, diskGB, secondaryDiskGB) > math.MaxInt32 {
		return &ErrorBundle{Err: fmt.Errorf("invalid"), UserMsg: "cores, ramGB, diskGB, secondaryDiskGB are too large", HttpCode: http.StatusBadRequest}
	}
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()
	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be edited", HttpCode: http.StatusBadRequest}
	}
	if rcr.CurrentSecondaryDiskGb > 0 && secondaryDiskGB < int(rcr.CurrentSecondaryDiskGb) {
		return &ErrorBundle{
			Err:      fmt.Errorf("HDD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink an existing %d GB HDD via edit.", rcr.CurrentSecondaryDiskGb),
			HttpCode: http.StatusBadRequest,
		}
	}
	if diskGB < int(rcr.CurrentDiskGb) {
		return &ErrorBundle{
			Err:      fmt.Errorf("SSD shrink not supported"),
			UserMsg:  fmt.Sprintf("Cannot shrink the %d GB SSD via edit.", rcr.CurrentDiskGb),
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
	edited := *rcr
	edited.NewCores, edited.NewRamGb, edited.NewDiskGb, edited.NewSecondaryDiskGb = int32(cores), int32(ramGB), int32(diskGB), int32(secondaryDiskGB)
	_ = notifier.NotifyResourceChangeRequestEdited(ctx, *rcr, edited)
	return nil
}

func RejectResourceChangeRequest(ctx context.Context, id int64, reason string) *ErrorBundle {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return &ErrorBundle{Err: fmt.Errorf("missing reason"), UserMsg: "A reason is required, it is sent to the VM owner.", HttpCode: http.StatusBadRequest}
	}
	if utf8.RuneCountInString(reason) > 1000 {
		return &ErrorBundle{Err: fmt.Errorf("reason too long"), UserMsg: "The reason must be at most 1000 characters.", HttpCode: http.StatusBadRequest}
	}
	rcr, unlock, eb := getLockedResourceChange(id)
	if eb != nil {
		return eb
	}
	defer unlock()
	if rcr.Status != storage.RcrStatusPending {
		return &ErrorBundle{Err: fmt.Errorf("not pending"), UserMsg: "Only pending resource change requests can be rejected", HttpCode: http.StatusBadRequest}
	}
	if err := storage.DB.MarkResourceChangeRequestProcessed(id, storage.RcrStatusRejected, reason); err != nil {
		return SimpleError(err, "Failed to update request status")
	}
	rcr.Status = storage.RcrStatusRejected
	_ = notifier.NotifyResourceChangeRequestStatusChanged(ctx, *rcr)
	if err := notifier.SendResourceChangeRejectedEmail(*rcr, vmOwnerContacts(rcr.VmHostname), reason); err != nil {
		log.Printf("Failed to send rejection email for request %d: %v", id, err)
	}
	return nil
}

// Loads the row twice: once to learn the hostname to lock, once under the lock.
func getLockedResourceChange(id int64) (*storage.ResourceChangeRequest, func(), *ErrorBundle) {
	ctx := context.Background()
	rcr, err := storage.DB.GetResourceChangeByID(ctx, id)
	if err != nil {
		return nil, nil, SimpleError(err, "Failed to fetch resource change request")
	}
	unlock, err := storage.DB.LockResourceChangeVM(rcr.VmHostname)
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

	r.Methods("GET").Path("/api/resourcechange/closure").HandlerFunc(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		closure, err := storage.DB.CurrentResourceChangeClosure(r.Context())
		if err != nil {
			log.Printf("Failed to get resource change closure: %v", err)
			http.Error(w, "Failed to get resource change closure", http.StatusInternalServerError)
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

	r.Methods("POST").Path("/api/resourcechange/close").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("close", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

		closure, err := storage.DB.CurrentResourceChangeClosure(r.Context())
		if err != nil {
			log.Printf("Failed to get resource change closure: %v", err)
			http.Error(w, "Failed to get resource change closure", http.StatusInternalServerError)
			return
		}
		if closure != nil {
			http.Error(w, "Resource change requests are already closed", http.StatusBadRequest)
			return
		}

		_, err = storage.DB.CreateResourceChangeClosure(r.Context(), body.Reason)
		if err != nil {
			log.Printf("Failed to close resource change requests: %v", err)
			http.Error(w, "Failed to close resource change requests", http.StatusInternalServerError)
			return
		}
		log.Printf("Resource change requests closed by %s: %s", auth.UserEmail(r.Context()), body.Reason)
		_ = notifier.NotifyResourceChangeClosure(r.Context(), "Resource change requests closed: "+body.Reason)
	}))))

	r.Methods("POST").Path("/api/resourcechange/reopen").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("reopen", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reopened, err := storage.DB.ReopenResourceChanges(r.Context())
		if err != nil {
			log.Printf("Failed to reopen resource change requests: %v", err)
			http.Error(w, "Failed to reopen resource change requests", http.StatusInternalServerError)
			return
		}
		if reopened == 0 {
			http.Error(w, "Resource change requests are not closed", http.StatusBadRequest)
			return
		}
		log.Printf("Resource change requests reopened by %s", auth.UserEmail(r.Context()))
		_ = notifier.NotifyResourceChangeClosure(r.Context(), "Resource change requests reopened")
	}))))

	// Submit endpoint, called by the SSH portal on behalf of an
	// authenticated VM owner. Authenticated via shared bearer token.
	r.Methods("POST").Path("/api/resourcechange").Handler(sshportalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rcrSubmitBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Printf("Error decoding resource change body: %v", err)
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}

		resp, eb := SubmitResourceChangeRequest(r.Context(), &body)
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
			ID     int    `json:"id"`
			Reason string `json:"reason"`
		}
		var b bodyS
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if eb := AcceptResourceChangeRequest(r.Context(), int64(b.ID), b.Reason); eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	// Admin: reject (confirmable — user must type "reject" to confirm)
	r.Methods("POST").Path("/api/resourcechange/reject").Subrouter().NewRoute().Handler(auth.CheckAuthenticated(confirmation.ConfirmMiddleware("reject", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type bodyS struct {
			ID     int    `json:"id"`
			Reason string `json:"reason"`
		}
		var b bodyS
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		if eb := RejectResourceChangeRequest(r.Context(), int64(b.ID), b.Reason); eb != nil {
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
		if eb := EditResourceChangeRequest(r.Context(), int64(b.ID), b.NewCores, b.NewRamGB, b.NewDiskGB, b.NewSecondaryDiskGB); eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
	}))))

	// SSH portal: cancel pending request for a VM
	r.Methods("POST").Path("/api/resourcechange/cancel").Handler(sshportalAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body rcrCancelBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "Invalid request payload", http.StatusBadRequest)
			return
		}
		rcr, eb := CancelResourceChangeRequest(r.Context(), body.VMHostname)
		if eb != nil {
			http.Error(w, eb.UserMsg, eb.HttpCode)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(toRcrJSON(*rcr))
	})))

}
