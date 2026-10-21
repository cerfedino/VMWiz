-- name: LockResourceChangeVM :exec
-- Serialize submit/cancel/undo/admin operations for one VM across instances.
SELECT pg_advisory_xact_lock(17017, hashtext(@vm_hostname::text));

-- name: LockResourceChangeSubmissions :exec
-- Serialize replacement even when no pending row exists yet.
SELECT pg_advisory_xact_lock(17018, 0);

-- name: SupersedePendingResourceChanges :exec
UPDATE resource_change_request
SET status = 'cancelled', processedAt = CURRENT_TIMESTAMP,
    processedNote = 'Superseded by a newer resource change request'
WHERE vmHostname = @vm_hostname AND status = 'pending';

-- name: CreateResourceChange :one
INSERT INTO resource_change_request (
    status, vmHostname, requesterFingerprint,
    currentCores, currentRamGB, currentDiskGB, currentSecondaryDiskGB,
    newCores, newRamGB, newDiskGB, newSecondaryDiskGB,
    needsReviewReasons, requestReason
) VALUES (
    @status, @vm_hostname, @requester_fingerprint,
    @current_cores, @current_ram_gb, @current_disk_gb, @current_secondary_disk_gb,
    @new_cores, @new_ram_gb, @new_disk_gb, @new_secondary_disk_gb,
    @needs_review_reasons, @request_reason
) RETURNING id;

-- name: GetResourceChangeByID :one
SELECT * FROM resource_change_request WHERE id = @id;

-- name: ListResourceChanges :many
SELECT * FROM resource_change_request ORDER BY createdAt DESC, id DESC;

-- name: GetLatestAppliedResourceChange :one
SELECT * FROM resource_change_request
WHERE vmHostname = @vm_hostname AND status IN ('auto_approved', 'accepted')
ORDER BY createdAt DESC, id DESC LIMIT 1;

-- name: GetPendingResourceChange :one
SELECT * FROM resource_change_request
WHERE vmHostname = @vm_hostname AND status = 'pending'
ORDER BY createdAt DESC, id DESC LIMIT 1;

-- name: UpdateResourceChangeTargets :execrows
UPDATE resource_change_request
SET newCores = @new_cores, newRamGB = @new_ram_gb,
    newDiskGB = @new_disk_gb, newSecondaryDiskGB = @new_secondary_disk_gb
WHERE id = @id AND status = 'pending';

-- name: SetResourceChangeOutcome :execrows
UPDATE resource_change_request
SET status = @status, processedAt = CURRENT_TIMESTAMP, processedNote = @note
WHERE id = @id AND status = 'pending';
