package reqcontext

import "context"

type AuthClaims struct {
	UserID int64
	Role   string
}

type authKey struct{}

func WithAuth(ctx context.Context, c AuthClaims) context.Context {
	return context.WithValue(ctx, authKey{}, c)
}

func AuthFromContext(ctx context.Context) (AuthClaims, bool) {
	c, ok := ctx.Value(authKey{}).(AuthClaims)
	return c, ok
}
