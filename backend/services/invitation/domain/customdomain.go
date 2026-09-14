package domain

import (
	"context"
	"net"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"

	"undangan/kernel/apperror"
)

// CustomDomain = domain milik customer yang diarahkan ke undangan (docs/SPEC.md §2 "Subdomain & custom domain").
type CustomDomain struct {
	Hostname    string
	VerifyToken string
	VerifiedAt  *time.Time
}

func (c *CustomDomain) Verified() bool { return c != nil && c.VerifiedAt != nil }

type DNSRecord struct {
	Type    string `json:"type"` // CNAME | A | TXT
	Name    string `json:"name"`
	Value   string `json:"value"`
	Purpose string `json:"purpose"`
}

// DomainConfig = target DNS platform.
type DomainConfig struct {
	BaseDomain  string
	CNAMETarget string   // mis. custom.undangin.id
	IPs         []string // IP server untuk A record domain apex (opsional)
}

// DNSResolver = port lookup DNS (implementasi: net.Resolver; bisa dipalsukan di test).
type DNSResolver interface {
	LookupTXT(ctx context.Context, name string) ([]string, error)
	LookupCNAME(ctx context.Context, host string) (string, error)
	LookupHost(ctx context.Context, host string) ([]string, error)
}

const verifyPrefix = "_undangan-verify."

// NormalizeHostname memvalidasi domain yang diinput customer: "https://WWW.Raka.com/" → "www.raka.com".
func NormalizeHostname(raw, baseDomain string) (string, error) {
	h := strings.ToLower(strings.TrimSpace(raw))
	h = strings.TrimPrefix(strings.TrimPrefix(h, "https://"), "http://")
	if i := strings.IndexAny(h, "/?#"); i >= 0 {
		h = h[:i]
	}
	h = strings.TrimSuffix(h, ".")
	invalid := apperror.Validation(map[string]string{"hostname": "Domain tidak valid. Contoh: rakadannadia.com atau www.rakadannadia.com"})
	if h == "" || len(h) > 253 || net.ParseIP(h) != nil || strings.Contains(h, ":") {
		return "", invalid
	}
	labels := strings.Split(h, ".")
	if len(labels) < 2 {
		return "", invalid
	}
	for _, l := range labels {
		if l == "" || len(l) > 63 || strings.HasPrefix(l, "-") || strings.HasSuffix(l, "-") {
			return "", invalid
		}
		for _, r := range l {
			if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
				return "", invalid
			}
		}
	}
	if etld, icann := publicsuffix.PublicSuffix(h); !icann || etld == h {
		return "", invalid // TLD tidak dikenal, atau hanya suffix (mis. "co.id")
	}
	base := strings.ToLower(baseDomain)
	if base != "" && (h == base || strings.HasSuffix(h, "."+base)) {
		return "", apperror.Validation(map[string]string{"hostname": "Gunakan fitur subdomain untuk alamat di domain platform"})
	}
	return h, nil
}

// IsApex: domain utama (rakadannadia.com / rakadannadia.co.id), bukan subdomain seperti www.
func IsApex(host string) bool {
	root, err := publicsuffix.EffectiveTLDPlusOne(host)
	return err == nil && root == host
}

// DNSInstructions = record yang harus dibuat customer di pengelola DNS domainnya.
func DNSInstructions(cd *CustomDomain, cfg DomainConfig) []DNSRecord {
	if cd == nil {
		return nil
	}
	recs := []DNSRecord{{Type: "TXT", Name: verifyPrefix + cd.Hostname, Value: cd.VerifyToken, Purpose: "Bukti kepemilikan domain"}}
	switch {
	case !IsApex(cd.Hostname):
		recs = append(recs, DNSRecord{Type: "CNAME", Name: cd.Hostname, Value: cfg.CNAMETarget, Purpose: "Mengarahkan domain ke undangan"})
	case len(cfg.IPs) > 0:
		for _, ip := range cfg.IPs {
			recs = append(recs, DNSRecord{Type: "A", Name: cd.Hostname, Value: ip, Purpose: "Mengarahkan domain utama ke server undangan"})
		}
	default:
		recs = append(recs, DNSRecord{Type: "CNAME", Name: cd.Hostname, Value: cfg.CNAMETarget,
			Purpose: "Domain utama: gunakan CNAME flattening / ALIAS (mis. di Cloudflare), atau pakai www."})
	}
	return recs
}

// CheckDNS memeriksa TXT verifikasi dan arah domain. Mengembalikan daftar yang belum terpenuhi.
func CheckDNS(ctx context.Context, r DNSResolver, cd *CustomDomain, cfg DomainConfig) []string {
	var missing []string

	txts, _ := r.LookupTXT(ctx, verifyPrefix+cd.Hostname)
	found := false
	for _, t := range txts {
		if strings.TrimSpace(t) == cd.VerifyToken {
			found = true
			break
		}
	}
	if !found {
		missing = append(missing, "TXT "+verifyPrefix+cd.Hostname+" belum berisi kode verifikasi")
	}

	target := strings.TrimSuffix(strings.ToLower(cfg.CNAMETarget), ".")
	if cname, err := r.LookupCNAME(ctx, cd.Hostname); err == nil && strings.TrimSuffix(strings.ToLower(cname), ".") == target {
		return missing
	}
	// Tanpa CNAME langsung (apex / CNAME flattening): IP domain harus sama dengan IP server/target.
	hostIPs, _ := r.LookupHost(ctx, cd.Hostname)
	allowed := map[string]bool{}
	for _, ip := range cfg.IPs {
		allowed[ip] = true
	}
	if targetIPs, err := r.LookupHost(ctx, target); err == nil {
		for _, ip := range targetIPs {
			allowed[ip] = true
		}
	}
	for _, ip := range hostIPs {
		if allowed[ip] {
			return missing
		}
	}
	return append(missing, cd.Hostname+" belum mengarah ke server undangan (CNAME/A record)")
}
