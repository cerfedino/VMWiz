package notifier

import (
	"context"
	"fmt"
	"log"

	"git.sos.ethz.ch/vsos/vmwiz.vsos.ethz.ch/vmwiz-backend/storage"
)

func NotifyResourceChangeRequest(req storage.SQLResourceChangeRequest) error {
	review := "eligible for automatic approval"
	if len(req.NeedsReviewReasons) > 0 {
		review = "needs admin review"
	}
	return notifyResourceChange("resourcechange_new", fmt.Sprintf("New resource change request %d (%s). Replaces any earlier pending request for this VM.\n```\n%v\n```", req.ID, review, req.ToString()))
}

func NotifyResourceChangeRequestApplied(req storage.SQLResourceChangeRequest, auto bool) error {
	prefix := "approved"
	if auto {
		prefix = "auto-approved"
	}
	return notifyResourceChange("resourcechange_accepted", fmt.Sprintf("Resource change request %d %s and applied: %s -> cores=%d ram=%dGB SSD=%dGB HDD=%dGB", req.ID, prefix, req.VMHostname, req.NewCores, req.NewRamGB, req.NewDiskGB, req.NewSecondaryDiskGB))
}

func NotifyResourceChangeRequestFailed(req storage.SQLResourceChangeRequest, errMsg string) error {
	return notifyResourceChange("resourcechange_failed", fmt.Sprintf("Resource change request %d failed to apply for %s: %s", req.ID, req.VMHostname, errMsg))
}

func NotifyResourceChangeRequestStatusChanged(req storage.SQLResourceChangeRequest) error {
	tag := "resourcechange_rejected"
	if req.Status == storage.RCR_STATUS_CANCELLED {
		tag = "resourcechange_cancelled"
	}
	return notifyResourceChange(tag, fmt.Sprintf("Resource change request %d for %s: %s", req.ID, req.VMHostname, req.Status))
}

func notifyResourceChange(tag, body string) error {
	err := useNotifier(context.Background(), tag, body)
	if err != nil {
		log.Printf("Resource change notification (%s) failed: %v", tag, err)
	}
	return err
}

// SendResourceChangeAppliedEmail emails the VM owner that a resource change has
// been applied to their VM. The change is staged in Proxmox; for memory and
// core changes the VM must reboot before the new specs take effect.
func SendResourceChangeAppliedEmail(req storage.SQLResourceChangeRequest, contacts []string, autoApproved bool) error {
	if len(contacts) == 0 {
		return nil
	}
	verb := "approved by an admin"
	if autoApproved {
		verb = "automatically approved"
	}
	subject := fmt.Sprintf("Resource change applied to %s", req.VMHostname)
	body := fmt.Sprintf(`Your resource change request for %s has been %s.

  cores:    %d -> %d
  RAM:      %d GB -> %d GB
  SSD disk: %d GB -> %d GB
  HDD disk: %d GB -> %d GB

The new configuration is staged in Proxmox. Memory and CPU changes only
take effect after the VM reboots. If you do not want the change to take effect,
please undo the request now via the SSH portal.

Reason given: %s
`, req.VMHostname, verb,
		req.CurrentCores, req.NewCores,
		req.CurrentRamGB, req.NewRamGB,
		req.CurrentDiskGB, req.NewDiskGB,
		req.CurrentSecondaryDiskGB, req.NewSecondaryDiskGB,
		req.RequestReason)
	return SendEmail(subject, []byte(body), contacts)
}
