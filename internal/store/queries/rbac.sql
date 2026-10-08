-- name: GetUserRoles :many
SELECT r.id, r.name
FROM roles r
JOIN user_roles ur ON ur.role_id = r.id
WHERE ur.user_id = $1 AND r.is_active = true;

-- name: GetUserPermissions :many
SELECT DISTINCT p.resource, p.action
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
JOIN user_roles ur ON ur.role_id = rp.role_id
WHERE ur.user_id = $1 AND p.is_active = true;

-- name: GetRoleByName :one
SELECT id, created_at, updated_at, is_active, name
FROM roles
WHERE name = $1;

-- name: ListRoles :many
SELECT id, created_at, updated_at, is_active, name
FROM roles
WHERE is_active = true
ORDER BY name;

-- name: ListPermissionsByRole :many
SELECT p.id, p.resource, p.action
FROM permissions p
JOIN role_permissions rp ON rp.permission_id = p.id
WHERE rp.role_id = $1 AND p.is_active = true
ORDER BY p.resource, p.action;

-- name: AssignRoleToUser :exec
INSERT INTO user_roles (user_id, role_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveRoleFromUser :exec
DELETE FROM user_roles
WHERE user_id = $1 AND role_id = $2;

-- name: CreatePermission :one
INSERT INTO permissions (resource, action)
VALUES ($1, $2)
ON CONFLICT (resource, action) DO UPDATE SET updated_at = now()
RETURNING id, created_at, updated_at, is_active, resource, action;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemovePermissionFromRole :exec
DELETE FROM role_permissions
WHERE role_id = $1 AND permission_id = $2;
