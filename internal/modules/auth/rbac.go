package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store/db"
)

type permCacheKey struct{}

type permSet map[[2]string]struct{}

type RBACService struct {
	q *db.Queries
}

func NewRBACService(pool *pgxpool.Pool) *RBACService {
	return &RBACService{q: db.New(pool)}
}

func (s *RBACService) HasPermission(ctx context.Context, userID int64, resource, action string) (bool, error) {
	roles, err := s.q.GetUserRoles(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, r := range roles {
		if r.Name == RoleAdmin {
			return true, nil
		}
	}

	perms, err := s.loadPerms(ctx, userID)
	if err != nil {
		return false, err
	}
	_, ok := perms[[2]string{resource, action}]
	return ok, nil
}

func (s *RBACService) loadPerms(ctx context.Context, userID int64) (permSet, error) {
	if cached, ok := ctx.Value(permCacheKey{}).(permSet); ok {
		return cached, nil
	}
	rows, err := s.q.GetUserPermissions(ctx, userID)
	if err != nil {
		return nil, err
	}
	ps := make(permSet, len(rows))
	for _, r := range rows {
		ps[[2]string{r.Resource, r.Action}] = struct{}{}
	}
	return ps, nil
}

func (s *RBACService) ListRoles(ctx context.Context) ([]db.Role, error) {
	return s.q.ListRoles(ctx)
}

func (s *RBACService) AssignRole(ctx context.Context, userID int64, roleName string) error {
	role, err := s.q.GetRoleByName(ctx, roleName)
	if err != nil {
		return err
	}
	return s.q.AssignRoleToUser(ctx, db.AssignRoleToUserParams{UserID: userID, RoleID: role.ID})
}

func (s *RBACService) RemoveRole(ctx context.Context, userID, roleID int64) error {
	return s.q.RemoveRoleFromUser(ctx, db.RemoveRoleFromUserParams{UserID: userID, RoleID: roleID})
}

func (s *RBACService) UserPermissions(ctx context.Context, userID int64) ([]db.GetUserPermissionsRow, error) {
	return s.q.GetUserPermissions(ctx, userID)
}

func (s *RBACService) UserRoles(ctx context.Context, userID int64) ([]db.GetUserRolesRow, error) {
	return s.q.GetUserRoles(ctx, userID)
}
