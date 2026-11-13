package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

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
func (s *postgresstorage) StoreResourceChangeRequest(p CreateResourceChangeParams) (int64, error) {
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
	if err := q.SupersedePendingResourceChanges(ctx, p.VmHostname); err != nil {
		return 0, err
	}
	// The column is NOT NULL; a nil slice would be sent as NULL.
	if p.NeedsReviewReasons == nil {
		p.NeedsReviewReasons = []string{}
	}
	id, err := q.CreateResourceChange(ctx, p)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// Returns nil without error when the VM has no applied request yet.
func (s *postgresstorage) GetLatestProcessedResourceChangeRequestForHostname(hostname string) (*ResourceChangeRequest, error) {
	row, err := s.Queries.GetLatestAppliedResourceChange(context.Background(), hostname)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// Returns nil without error when the VM has no pending request.
func (s *postgresstorage) GetLatestPendingResourceChangeRequestForHostname(hostname string) (*ResourceChangeRequest, error) {
	row, err := s.Queries.GetPendingResourceChange(context.Background(), hostname)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *postgresstorage) UpdateResourceChangeRequestNewValues(p UpdateResourceChangeTargetsParams) error {
	count, err := s.Queries.UpdateResourceChangeTargets(context.Background(), p)
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("resource change request is no longer pending")
	}
	return nil
}

func (s *postgresstorage) MarkResourceChangeRequestProcessed(id int64, status RcrStatus, note string) error {
	count, err := s.Queries.SetResourceChangeOutcome(context.Background(), SetResourceChangeOutcomeParams{ID: id, Status: status, Note: note})
	if err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("resource change request is no longer pending")
	}
	return nil
}
