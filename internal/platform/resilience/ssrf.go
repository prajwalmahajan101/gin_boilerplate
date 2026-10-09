package resilience

import (
	"net"
	"net/url"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/platform/apperr"
)

// AssertPublicURL rejects URLs that would let an outbound request reach the
// server's own network: a non-HTTP(S) scheme, a missing host, or a host that
// resolves to any loopback / private / link-local / unspecified / multicast
// address. Use it to guard SSRF-prone outbound calls (webhooks, user-supplied
// URLs) before dialing.
//
// ponytail: TOCTOU ceiling — DNS can rebind between this check and the actual
// dial, so a resolved-and-approved host could point elsewhere at connect time.
// Adequate for a boilerplate guard; the airtight fix is a Dialer.Control hook
// that re-checks the concrete IP at connect time. Upgrade there if this guards
// a real untrusted-URL path.
func AssertPublicURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return apperr.ValidationError("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return apperr.ValidationError("url scheme must be http or https")
	}
	host := u.Hostname()
	if host == "" {
		return apperr.ValidationError("url missing host")
	}

	// A literal IP needs no lookup; a hostname is resolved to every address it
	// maps to and all of them must be public.
	var ips []net.IP
	if ip := net.ParseIP(host); ip != nil {
		ips = []net.IP{ip}
	} else {
		resolved, lookupErr := net.LookupIP(host)
		if lookupErr != nil {
			return apperr.ValidationError("url host does not resolve")
		}
		ips = resolved
	}

	for _, ip := range ips {
		if !isPublicIP(ip) {
			return apperr.ValidationError("url resolves to a non-public address")
		}
	}
	return nil
}

// isPublicIP reports whether ip is globally routable — not loopback, private,
// link-local, unspecified, or multicast.
func isPublicIP(ip net.IP) bool {
	switch {
	case ip.IsLoopback(),
		ip.IsPrivate(),
		ip.IsLinkLocalUnicast(),
		ip.IsLinkLocalMulticast(),
		ip.IsUnspecified(),
		ip.IsMulticast():
		return false
	default:
		return true
	}
}
