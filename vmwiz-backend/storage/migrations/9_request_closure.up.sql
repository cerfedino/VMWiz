CREATE TABLE request_closure (
  id          BIGSERIAL PRIMARY KEY,
  closed_at   TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  reopened_at TIMESTAMP WITH TIME ZONE, -- NULL while closed
  reason      TEXT NOT NULL
);

-- At most one open closure at a time
CREATE UNIQUE INDEX request_closure_open ON request_closure ((reopened_at IS NULL)) WHERE reopened_at IS NULL;

CREATE TABLE request_waitlist (
  id         BIGSERIAL PRIMARY KEY,
  closure_id BIGINT NOT NULL REFERENCES request_closure(id),
  email      TEXT NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (closure_id, email)
);
