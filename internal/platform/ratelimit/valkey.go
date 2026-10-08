package ratelimit

import (
	"context"
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/valkey"
)

type ValkeyLimiter struct {
	vk *valkey.Client
}

func NewValkeyLimiter(vk *valkey.Client) *ValkeyLimiter {
	return &ValkeyLimiter{vk: vk}
}

func (v *ValkeyLimiter) Allow(ctx context.Context, key string, rpm int) (bool, error) {
	count, err := v.vk.Incr(ctx, key)
	if err != nil {
		return false, err
	}
	if count == 1 {
		_ = v.vk.Expire(ctx, key, time.Minute)
	}
	return count <= int64(rpm), nil
}
