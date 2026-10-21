package storage

import (
	"database/sql"
	"fmt"
	"time"
)

const (
	RCR_STATUS_PENDING       = "pending"
	RCR_STATUS_AUTO_APPROVED = "auto_approved"
	RCR_STATUS_ACCEPTED      = "accepted"
	RCR_STATUS_REJECTED      = "rejected"
	RCR_STATUS_FAILED        = "failed"
	RCR_STATUS_CANCELLED     = "cancelled"
)

type SQLResourceChangeRequest struct {
	ID                     int64        `db:"id" json:"ID"`
	CreatedAt              time.Time    `db:"createdAt" json:"CreatedAt"`
	Status                 string       `db:"status" json:"Status"`
	VMHostname             string       `db:"vmHostname" json:"VMHostname"`
	RequesterFingerprint   string       `db:"requesterFingerprint" json:"RequesterFingerprint"`
	CurrentCores           int          `db:"currentCores" json:"CurrentCores"`
	CurrentRamGB           int          `db:"currentRamGB" json:"CurrentRamGB"`
	CurrentDiskGB          int          `db:"currentDiskGB" json:"CurrentDiskGB"`
	CurrentSecondaryDiskGB int          `db:"currentSecondaryDiskGB" json:"CurrentSecondaryDiskGB"`
	NewCores               int          `db:"newCores" json:"NewCores"`
	NewRamGB               int          `db:"newRamGB" json:"NewRamGB"`
	NewDiskGB              int          `db:"newDiskGB" json:"NewDiskGB"`
	NewSecondaryDiskGB     int          `db:"newSecondaryDiskGB" json:"NewSecondaryDiskGB"`
	NeedsReviewReasons     []string     `db:"needsReviewReasons" json:"NeedsReviewReasons"`
	ProcessedAt            sql.NullTime `db:"processedAt" json:"-"`
	ProcessedNote          string       `db:"processedNote" json:"ProcessedNote"`
	RequestReason          string       `db:"requestReason" json:"RequestReason"`
}

func (r *SQLResourceChangeRequest) ToString() string {
	return fmt.Sprintf(`ID: %d
CreatedAt: %v
Status: %s
VMHostname: %s
RequesterFingerprint: %s
Cores: %d -> %d
RAM (GB): %d -> %d
Disk (GB): %d -> %d
Secondary HDD (GB): %d -> %d
Reason: %s
NeedsReviewReasons: %v
ProcessedNote: %s
`, r.ID, r.CreatedAt, r.Status, r.VMHostname, r.RequesterFingerprint, r.CurrentCores, r.NewCores, r.CurrentRamGB, r.NewRamGB, r.CurrentDiskGB, r.NewDiskGB, r.CurrentSecondaryDiskGB, r.NewSecondaryDiskGB, r.RequestReason, r.NeedsReviewReasons, r.ProcessedNote)
}
