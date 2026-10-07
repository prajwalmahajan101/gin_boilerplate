-- name: GetItemByCode :one
SELECT id, created_at, updated_at, is_active, name, code, notes
FROM items
WHERE code = $1;
