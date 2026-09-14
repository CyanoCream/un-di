package domain

import (
	"context"
	"errors"
	"testing"
)

func TestNormalizeHostname(t *testing.T) {
	ok := map[string]string{
		"https://WWW.RakaDanNadia.com/": "www.rakadannadia.com",
		"rakadannadia.co.id":            "rakadannadia.co.id",
		"undangan.keluarga-budi.my.id.": "undangan.keluarga-budi.my.id",
	}
	for in, want := range ok {
		got, err := NormalizeHostname(in, "undangin.id")
		if err != nil || got != want {
			t.Errorf("NormalizeHostname(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "localhost", "co.id", "10.0.0.1", "raka.undangin.id", "undangin.id", "-a.com", "a..com", "raka.invalidtld", "ra ka.com"} {
		if _, err := NormalizeHostname(bad, "undangin.id"); err == nil {
			t.Errorf("NormalizeHostname(%q) should fail", bad)
		}
	}
}

func TestDNSInstructions(t *testing.T) {
	cfg := DomainConfig{CNAMETarget: "custom.undangin.id", IPs: []string{"203.0.113.10"}}
	www := DNSInstructions(&CustomDomain{Hostname: "www.raka.com", VerifyToken: "tok"}, cfg)
	if len(www) != 2 || www[0].Type != "TXT" || www[0].Name != "_undangan-verify.www.raka.com" || www[1].Type != "CNAME" {
		t.Errorf("www = %+v", www)
	}
	apex := DNSInstructions(&CustomDomain{Hostname: "raka.co.id", VerifyToken: "tok"}, cfg)
	if len(apex) != 2 || apex[1].Type != "A" || apex[1].Value != "203.0.113.10" {
		t.Errorf("apex = %+v", apex)
	}
}

type fakeDNS struct {
	txt   map[string][]string
	cname map[string]string
	hosts map[string][]string
}

var errNX = errors.New("no such host")

func (f fakeDNS) LookupTXT(_ context.Context, n string) ([]string, error) {
	if v, ok := f.txt[n]; ok {
		return v, nil
	}
	return nil, errNX
}
func (f fakeDNS) LookupCNAME(_ context.Context, n string) (string, error) {
	if v, ok := f.cname[n]; ok {
		return v, nil
	}
	return "", errNX
}
func (f fakeDNS) LookupHost(_ context.Context, n string) ([]string, error) {
	if v, ok := f.hosts[n]; ok {
		return v, nil
	}
	return nil, errNX
}

func TestCheckDNS(t *testing.T) {
	cfg := DomainConfig{CNAMETarget: "custom.undangin.id"}
	cd := &CustomDomain{Hostname: "www.raka.com", VerifyToken: "undangan-verify=abc"}
	ctx := context.Background()

	if m := CheckDNS(ctx, fakeDNS{}, cd, cfg); len(m) != 2 {
		t.Errorf("empty DNS missing = %v", m)
	}
	ready := fakeDNS{
		txt:   map[string][]string{"_undangan-verify.www.raka.com": {"other", "undangan-verify=abc"}},
		cname: map[string]string{"www.raka.com": "Custom.Undangin.ID."},
	}
	if m := CheckDNS(ctx, ready, cd, cfg); len(m) != 0 {
		t.Errorf("ready missing = %v", m)
	}
	// Apex dengan CNAME flattening: IP sama dengan IP target.
	flat := fakeDNS{
		txt:   map[string][]string{"_undangan-verify.raka.com": {"undangan-verify=abc"}},
		hosts: map[string][]string{"raka.com": {"198.51.100.7"}, "custom.undangin.id": {"198.51.100.7"}},
	}
	if m := CheckDNS(ctx, flat, &CustomDomain{Hostname: "raka.com", VerifyToken: "undangan-verify=abc"}, cfg); len(m) != 0 {
		t.Errorf("flattened missing = %v", m)
	}
	wrongIP := fakeDNS{
		txt:   map[string][]string{"_undangan-verify.raka.com": {"undangan-verify=abc"}},
		hosts: map[string][]string{"raka.com": {"192.0.2.1"}, "custom.undangin.id": {"198.51.100.7"}},
	}
	if m := CheckDNS(ctx, wrongIP, &CustomDomain{Hostname: "raka.com", VerifyToken: "undangan-verify=abc"}, cfg); len(m) != 1 {
		t.Errorf("wrong IP missing = %v", m)
	}
}
