-- name: CreateMessage :exec
INSERT INTO messages (id, tenant_id, payload)
VALUES ($1, $2, $3);

-- name: GetMessagesByTenant :many
SELECT id, tenant_id, payload, created_at
FROM messages
WHERE tenant_id = $1
ORDER BY created_at DESC;
