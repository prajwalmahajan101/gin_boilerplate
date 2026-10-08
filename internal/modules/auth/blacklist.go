package auth

import (
	"context"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

type Blacklist struct {
	vk *valkey.Client
}

func NewBlacklist(vk *valkey.Client) *Blacklist {
	return &Blacklist{vk: vk}
}

func (b *Blacklist) Add(ctx context.Context, jti string, ttl time.Duration) error {
	if b == nil || b.vk == nil {
		return nil
	}
	return b.vk.SetEX(ctx, "bl:"+jti, "1", ttl)
}

func (b *Blacklist) IsBlacklisted(ctx context.Context, jti string) bool {
	if b == nil || b.vk == nil {
		return false
	}
	v, _ := b.vk.Get(ctx, "bl:"+jti)
	return v != ""
}
