package tenant

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		host string
		kind Kind
		name string
	}{
		{"undangin.id", KindPlatform, ""},
		{"www.undangin.id", KindPlatform, ""},
		{"api.undangin.id", KindAPI, ""},
		{"budi-ani.undangin.id", KindSubdomain, "budi-ani"},
		{"Budi-Ani.Undangin.id:443", KindSubdomain, "budi-ani"},
		{"admin.undangin.id", KindAdminApp, ""},
		{"app.undangin.id", KindPortalApp, ""},
		{"mail.undangin.id", KindUnknown, ""},
		{"a.b.undangin.id", KindUnknown, ""},
		{"budiani.com", KindCustomDomain, "budiani.com"},
		{"www.budiani.com.", KindCustomDomain, "www.budiani.com"},
	}
	for _, c := range cases {
		got := Parse(c.host, "undangin.id")
		if got.Kind != c.kind || got.Name != c.name {
			t.Errorf("Parse(%q) = %+v, want kind=%d name=%q", c.host, got, c.kind, c.name)
		}
	}
}

func TestValidSubdomain(t *testing.T) {
	for label, want := range map[string]bool{
		"budi-ani": true, "ab": false, "-budi": false, "budi-": false,
		"budi_ani": false, "admin": false, "rizky123": true,
	} {
		if got := ValidSubdomain(label); got != want {
			t.Errorf("ValidSubdomain(%q) = %v, want %v", label, got, want)
		}
	}
}
