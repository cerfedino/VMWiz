CREATE TABLE log_scope (
  id          TEXT PRIMARY KEY,
  parent_id   TEXT REFERENCES log_scope(id) ON DELETE CASCADE,
  root_id     TEXT NOT NULL, -- top-level scope owning the log file <root_id>.log
  label       TEXT NOT NULL DEFAULT '',
  started_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ended_at    TIMESTAMP WITH TIME ZONE, -- NULL while open
  failed      BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_log_scope_parent ON log_scope(parent_id);
CREATE INDEX idx_log_scope_root ON log_scope(root_id);
CREATE INDEX idx_log_scope_retention ON log_scope(ended_at);

-- Reserved catch-all scope '0', owns 0.log, never swept by retention.
INSERT INTO log_scope (id, parent_id, root_id, label)
VALUES ('0', NULL, '0', 'catch-all');
