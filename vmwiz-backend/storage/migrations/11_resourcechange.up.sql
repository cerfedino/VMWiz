CREATE TYPE rcr_status AS ENUM ('pending', 'auto_approved', 'accepted', 'rejected', 'failed', 'cancelled');

CREATE TABLE resource_change_request (
  id                        BIGSERIAL PRIMARY KEY,
  created_at                TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  status                    rcr_status NOT NULL DEFAULT 'pending',
  vm_hostname               TEXT NOT NULL,
  requester_fingerprint     TEXT NOT NULL DEFAULT '',
  current_cores             INT NOT NULL,
  current_ram_gb            INT NOT NULL,
  current_disk_gb           INT NOT NULL,
  new_cores                 INT NOT NULL,
  new_ram_gb                INT NOT NULL,
  new_disk_gb               INT NOT NULL,
  needs_review_reasons      TEXT[] NOT NULL DEFAULT '{}',
  processed_at              TIMESTAMP WITH TIME ZONE,
  processed_note            TEXT NOT NULL DEFAULT '',
  request_reason            TEXT NOT NULL DEFAULT '',
  current_secondary_disk_gb INT NOT NULL DEFAULT 0,
  new_secondary_disk_gb     INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_resource_change_request_hostname_created
    ON resource_change_request (vm_hostname, created_at DESC);

CREATE UNIQUE INDEX resource_change_request_one_pending_per_vm
    ON resource_change_request (vm_hostname) WHERE status = 'pending';
