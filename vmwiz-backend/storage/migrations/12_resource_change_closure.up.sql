ALTER TABLE request_closure RENAME TO vm_request_closure;
ALTER INDEX request_closure_open RENAME TO vm_request_closure_open;
ALTER TABLE request_waitlist RENAME TO vm_request_waitlist;

CREATE TABLE resource_change_closure (
  id          BIGSERIAL PRIMARY KEY,
  closed_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  reopened_at TIMESTAMP WITH TIME ZONE, -- NULL while closed
  reason      TEXT NOT NULL
);

-- At most one open closure at a time
CREATE UNIQUE INDEX resource_change_closure_open ON resource_change_closure ((reopened_at IS NULL)) WHERE reopened_at IS NULL;
