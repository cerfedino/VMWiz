package storage

import (
	"context"
	"database/sql"
	"fmt"
	"math"
)

func resourceChangeFromRow(r ResourceChangeRequest) *SQLResourceChangeRequest {
	return &SQLResourceChangeRequest{
		ID:                     r.ID,
		CreatedAt:              r.Createdat,
		Status:                 string(r.Status),
		VMHostname:             r.Vmhostname,
		RequesterFingerprint:   r.Requesterfingerprint,
		CurrentCores:           int(r.Currentcores),
		CurrentRamGB:           int(r.Currentramgb),
		CurrentDiskGB:          int(r.Currentdiskgb),
		CurrentSecondaryDiskGB: int(r.Currentsecondarydiskgb),
		NewCores:               int(r.Newcores),
		NewRamGB:               int(r.Newramgb),
		NewDiskGB:              int(r.Newdiskgb),
		NewSecondaryDiskGB:     int(r.Newsecondarydiskgb),
		NeedsReviewReasons:     r.Needsreviewreasons,
		ProcessedAt:            r.Processedat,
		ProcessedNote:          r.Processednote,
		RequestReason:          r.Requestreason,
	}
}

// LockResourceChangeVM holds a database advisory lock until release is called.
// Keep the lock across the VM write and status update, so an older request
// cannot be accepted while a new submission supersedes it.
func (s *postgresstorage) LockResourceChangeVM(hostname string) (release func(), err error) {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return nil, err
	}
	if err := s.Queries.WithTx(tx).LockResourceChangeVM(context.Background(), hostname); err != nil {
		tx.Rollback()
		return nil, err
	}
	return func() { _ = tx.Rollback() }, nil
}

// StoreResourceChangeRequest replaces all pending requests for this VM in the
// same transaction as the new insert. History remains available as cancelled.
func (s *postgresstorage) StoreResourceChangeRequest(req *SQLResourceChangeRequest) (int64, error) {
	for _, value := range []int{req.CurrentCores, req.CurrentRamGB, req.CurrentDiskGB, req.CurrentSecondaryDiskGB, req.NewCores, req.NewRamGB, req.NewDiskGB, req.NewSecondaryDiskGB} {
		if int64(value) > math.MaxInt32 || int64(value) < math.MinInt32 {
			return 0, fmt.Errorf("resource value out of database range")
		}
	}
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	q := s.Queries.WithTx(tx)
	if err := q.LockResourceChangeSubmissions(ctx); err != nil {
		return 0, err
	}
	if err := q.SupersedePendingResourceChanges(ctx, req.VMHostname); err != nil {
		return 0, err
	}
	reasons := req.NeedsReviewReasons
	if reasons == nil {
		reasons = []string{}
	}
	id, err := q.CreateResourceChange(ctx, CreateResourceChangeParams{
		Status: RcrStatus(req.Status), VmHostname: req.VMHostname, RequesterFingerprint: req.RequesterFingerprint,
		CurrentCores: int32(req.CurrentCores), CurrentRamGb: int32(req.CurrentRamGB), CurrentDiskGb: int32(req.CurrentDiskGB), CurrentSecondaryDiskGb: int32(req.CurrentSecondaryDiskGB),
		NewCores: int32(req.NewCores), NewRamGb: int32(req.NewRamGB), NewDiskGb: int32(req.NewDiskGB), NewSecondaryDiskGb: int32(req.NewSecondaryDiskGB),
		NeedsReviewReasons: reasons, RequestReason: req.RequestReason,
	})
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *postgresstorage) GetResourceChangeRequestById(id int64) (*SQLResourceChangeRequest, error) {
	row, err := s.Queries.GetResourceChangeByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	return resourceChangeFromRow(row), nil
}

func (s *postgresstorage) GetAllResourceChangeRequests() ([]*SQLResourceChangeRequest, error) {
	rows, err := s.Queries.ListResourceChanges(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]*SQLResourceChangeRequest, 0, len(rows))
	for _, row := range rows {
		out = append(out, resourceChangeFromRow(row))
	}
	return out, nil
}

func (s *postgresstorage) GetLatestProcessedResourceChangeRequestForHostname(hostname string) (*SQLResourceChangeRequest, error) {
	row, err := s.Queries.GetLatestAppliedResourceChange(context.Background(), hostname)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return resourceChangeFromRow(row), nil
}

func (s *postgresstorage) GetLatestPendingResourceChangeRequestForHostname(hostname string) (*SQLResourceChangeRequest, error) {
	row, err := s.Queries.GetPendingResourceChange(context.Background(), hostname)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return resourceChangeFromRow(row), nil
}

func (s *postgresstorage) UpdateResourceChangeRequestNewValues(id int64, cores, ramGB, diskGB, secondaryDiskGB int) error {
	for _, value := range []int{cores, ramGB, diskGB, secondaryDiskGB} {
		if int64(value) > math.MaxInt32 || int64(value) < math.MinInt32 {
			return fmt.Errorf("resource value out of database range")
		}
	}
	count, err := s.Queries.UpdateResourceChangeTargets(context.Background(), UpdateResourceChangeTargetsParams{
		ID: id, NewCores: int32(cores), NewRamGb: int32(ramGB), NewDiskGb: int32(diskGB), NewSecondaryDiskGb: int32(secondaryDiskGB),
	})
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("resource change request is no longer pending")
	}
	return nil
}

func (s *postgresstorage) MarkResourceChangeRequestProcessed(id int64, status string, note string) error {
	count, err := s.Queries.SetResourceChangeOutcome(context.Background(), SetResourceChangeOutcomeParams{ID: id, Status: RcrStatus(status), Note: note})
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("resource change request is no longer pending")
	}
	return nil
}
