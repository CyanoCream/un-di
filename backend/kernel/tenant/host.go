// Package tenant memetakan Host header ke jenis tujuan request.
package tenant

import (
	"net"
	"regexp"
	"strings"
)

type Kind int

const (
	KindUnknown      Kind = iota
	KindPlatform          // apex / www → landing page marketing
	KindAPI               // api.<base>
	KindAdminApp          // admin.<base> → portal super admin (SPA)
	KindPortalApp         // app.<base> → portal customer (SPA)
	KindSubdomain         // <slug>.<base> → undangan
	KindCustomDomain      // domain milik customer → undangan
)

type Host struct {
	Kind Kind
	Name string // label subdomain (KindSubdomain) atau hostname lengkap (KindCustomDomain)
}

// Subdomain yang tidak boleh dipakai customer.
var reserved = map[string]bool{
	"www": true, "api": true, "app": true, "admin": true, "mail": true,
	"smtp": true, "static": true, "cdn": true, "assets": true, "custom": true,
	"status": true, "blog": true, "help": true, "support": true, "dev": true,
}

var labelRe = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$`)

// ValidSubdomain: 3–63 char, a-z 0-9 dan '-', tidak diawali/diakhiri '-', bukan reserved.
func ValidSubdomain(label string) bool {
	return len(label) >= 3 && labelRe.MatchString(label) && !reserved[label]
}

func Normalize(host string) string {
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func Parse(rawHost, baseDomain string) Host {
	host := Normalize(rawHost)
	base := Normalize(baseDomain)

	switch {
	case host == "":
		return Host{Kind: KindUnknown}
	case host == base || host == "www."+base:
		return Host{Kind: KindPlatform}
	case host == "api."+base:
		return Host{Kind: KindAPI}
	case host == "admin."+base:
		return Host{Kind: KindAdminApp}
	case host == "app."+base:
		return Host{Kind: KindPortalApp}
	case strings.HasSuffix(host, "."+base):
		label := strings.TrimSuffix(host, "."+base)
		// Hanya satu level (budi-ani.base), tolak a.b.base.
		if strings.Contains(label, ".") || reserved[label] {
			return Host{Kind: KindUnknown}
		}
		return Host{Kind: KindSubdomain, Name: label}
	default:
		return Host{Kind: KindCustomDomain, Name: host}
	}
}
