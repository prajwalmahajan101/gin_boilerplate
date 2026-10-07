package httpserver

import (
	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

// configureTrustedProxies tells gin which proxy addresses are allowed to set
// X-Forwarded-For. An empty list trusts no proxy, so c.ClientIP() falls back to
// the direct RemoteAddr — the safe default behind no proxy.
//
// ponytail: gin owns the CIDR/XFF parsing. Swap to a custom parser only if a
// proxy topology appears that gin's SetTrustedProxies cannot model.
func configureTrustedProxies(r *gin.Engine, cfg *config.Config) error {
	return r.SetTrustedProxies(cfg.TrustedProxies)
}

// ClientIP returns the resolved client IP, honoring the configured trusted
// proxies. Thin passthrough so call sites depend on this package, not gin.
func ClientIP(c *gin.Context) string {
	return c.ClientIP()
}
