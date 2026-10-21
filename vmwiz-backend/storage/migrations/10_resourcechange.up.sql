CREATE TYPE rcr_status AS ENUM ('pending', 'auto_approved', 'accepted', 'rejected', 'failed', 'cancelled');

CREATE TABLE resource_change_request (
  id BIGSERIAL PRIMARY KEY,
  createdAt TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status rcr_status NOT NULL DEFAULT 'pending',
  vmHostname text NOT NULL,
  requesterFingerprint text NOT NULL DEFAULT '',
  currentCores int NOT NULL,
  currentRamGB int NOT NULL,
  currentDiskGB int NOT NULL,
  newCores int NOT NULL,
  newRamGB int NOT NULL,
  newDiskGB int NOT NULL,
  needsReviewReasons text[] NOT NULL DEFAULT '{}',
  processedAt TIMESTAMP,
  processedNote text NOT NULL DEFAULT '',
  requestReason text NOT NULL DEFAULT '',
  currentSecondaryDiskGB int NOT NULL DEFAULT 0,
  newSecondaryDiskGB int NOT NULL DEFAULT 0
);

CREATE INDEX idx_resource_change_request_hostname_created
    ON resource_change_request (vmHostname, createdAt DESC);

CREATE UNIQUE INDEX resource_change_request_one_pending_per_vm
    ON resource_change_request (vmHostname) WHERE status = 'pending';
