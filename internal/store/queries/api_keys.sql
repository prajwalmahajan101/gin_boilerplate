-- name: GetAPIKeyByPrefix :one
SELECT ak.id, ak.created_at, ak.updated_at, ak.is_active,
       ak.user_id, ak.name, ak.prefix, ak.key_hash,
       ak.expires_at, ak.last_used_at,
       u.role AS user_role
FROM api_keys ak
JOIN users u ON u.id = ak.user_id AND u.is_active = true
WHERE ak.prefix = $1 AND ak.is_active = true;

-- name: ListAPIKeysByUserID :many
SELECT id, created_at, updated_at, is_active, user_id, name, prefix, key_hash, expires_at, last_used_at
FROM api_keys
WHERE user_id = $1 AND is_active = true
ORDER BY id;

-- name: TouchAPIKeyUsage :exec
UPDATE api_keys SET last_used_at = now(), updated_at = now()
WHERE id = $1;
