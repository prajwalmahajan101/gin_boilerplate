-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, is_active, email, password_hash, role, last_login_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, created_at, updated_at, is_active, email, password_hash, role, last_login_at
FROM users
WHERE id = $1;

-- name: TouchUserLogin :one
UPDATE users SET last_login_at = now(), updated_at = now()
WHERE id = $1
RETURNING id, created_at, updated_at, is_active, email, password_hash, role, last_login_at;
