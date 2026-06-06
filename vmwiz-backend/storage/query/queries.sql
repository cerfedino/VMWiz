-- name: CreateVMRequest :one
INSERT INTO request (
  email, personalEmail, isOrganization, orgName, hostname, image,
  cores, ramGB, diskGB, secondaryDiskGB, sshPubkeys, comments
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING requestID;

-- name: GetVMRequestByID :one
SELECT * FROM request WHERE requestID = $1;

-- name: GetVMRequestsByHostname :many
SELECT * FROM request WHERE hostname = $1;

-- name: ListVMRequests :many
SELECT * FROM request ORDER BY requestID;

-- name: UpdateVMRequest :exec
UPDATE request SET
  requestCreatedAt = $2,
  requestStatus = $3,
  email = $4,
  personalEmail = $5,
  isOrganization = $6,
  orgName = $7,
  hostname = $8,
  image = $9,
  cores = $10,
  ramGB = $11,
  diskGB = $12,
  secondaryDiskGB = $13,
  sshPubkeys = $14,
  comments = $15
WHERE requestID = $1;

-- name: UpdateVMRequestStatus :exec
UPDATE request SET requestStatus = $2 WHERE requestID = $1;





-- name: CreateSurvey :one
INSERT INTO survey DEFAULT VALUES RETURNING id;

-- name: GetSurveyByID :one
SELECT * FROM survey WHERE id = $1;

-- name: ListSurveys :many
SELECT * FROM survey ORDER BY id;

-- name: ListSurveyIDs :many
SELECT id FROM survey ORDER BY id;

-- name: GetLatestSurveyID :one
SELECT id FROM survey ORDER BY date DESC LIMIT 1;







-- name: CreateLogScope :exec
INSERT INTO log_scope (id, parent_id, root_id, label) VALUES ($1, $2, $3, $4);

-- name: FinishLogScope :exec
UPDATE log_scope SET ended_at = CURRENT_TIMESTAMP, failed = $2 WHERE id = $1;

-- name: GetLogScopeStatus :one
SELECT ended_at, failed FROM log_scope WHERE id = $1;

-- name: GetLogScopeRootID :one
SELECT root_id FROM log_scope WHERE id = $1;

-- name: ListLogScopeSubtreeIDs :many
WITH RECURSIVE subtree AS (
  SELECT log_scope.id FROM log_scope WHERE log_scope.id = $1
  UNION ALL
  SELECT c.id FROM log_scope c JOIN subtree st ON c.parent_id = st.id
)
SELECT subtree.id FROM subtree;


-- name: ListRootLogScopes :many
-- Top-level scopes (id = root_id) newest-first. IDs are time-ordered, so before_id pages to scopes created before it; pass '' to start from newest.
SELECT * FROM log_scope
WHERE id = root_id
  AND id <> sqlc.arg(root_scope_id)
  AND (sqlc.arg(before_id)::text = '' OR id < sqlc.arg(before_id)::text)
ORDER BY id DESC
LIMIT sqlc.arg(max_results);

-- name: ListExpiredRootLogScopeIDs :many
SELECT id FROM log_scope
WHERE id = root_id
  AND id <> sqlc.arg(root_scope_id)
  AND ended_at IS NOT NULL
  AND ended_at < sqlc.arg(cutoff);





-- name: CreateSurveyEmail :one
INSERT INTO survey_email (
  recipient, surveyId, vmid, hostname, uuid, email_sent, still_used
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING id;

-- name: UpdateSurveyEmailResponse :exec
UPDATE survey_email SET still_used = $2 WHERE uuid = $1;

-- name: MarkSurveyEmailSent :exec
UPDATE survey_email SET email_sent = TRUE WHERE uuid = $1;

-- name: ListUnansweredOrUnsentSurveyEmails :many
SELECT * FROM survey_email
WHERE surveyId = $1 AND (still_used IS NULL OR email_sent = FALSE);

-- name: ListSentUnansweredSurveyEmails :many
SELECT * FROM survey_email
WHERE surveyId = $1 AND (still_used IS NULL AND email_sent = TRUE);

-- name: ListUnsentSurveyEmails :many
SELECT * FROM survey_email
WHERE surveyId = $1 AND (email_sent = FALSE);

-- name: CountUnsentSurveyEmails :one
SELECT COUNT(*) FROM survey_email
WHERE surveyId = $1 AND (email_sent = FALSE);

-- name: CountPositiveSurveyEmails :one
SELECT COUNT(*) FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used = TRUE);

-- name: CountNegativeSurveyEmails :one
SELECT COUNT(*) FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used = FALSE);

-- name: CountUnansweredSurveyEmails :one
SELECT COUNT(*) FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used IS NULL);

-- name: ListPositiveSurveyHostnames :many
SELECT hostname FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used = TRUE);

-- name: ListNegativeSurveyHostnames :many
SELECT hostname FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used = FALSE);

-- name: ListUnansweredSurveyHostnames :many
SELECT hostname FROM survey_email
WHERE surveyId = $1 AND (email_sent = TRUE AND still_used IS NULL);

-- name: ListUnsentSurveyHostnames :many
SELECT hostname FROM survey_email
WHERE surveyId = $1 AND email_sent = FALSE;

-- name: SurveyEmailExistsByUUID :one
SELECT EXISTS(SELECT 1 FROM survey_email WHERE uuid = $1);
