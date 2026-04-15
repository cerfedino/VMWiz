CREATE TABLE operation_log(
  logID BIGSERIAL PRIMARY KEY,
  operationID VARCHAR(255) NOT NULL,
  timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
  message TEXT NOT NULL
);

CREATE INDEX idx_oplog_opid ON operation_log(operationID);
