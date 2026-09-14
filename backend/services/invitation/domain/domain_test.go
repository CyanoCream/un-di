package domain

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Bapak Joko & Keluarga":       "bapak-joko-keluarga",
		"  Ibu Siti Nurhaliza, S.Pd ": "ibu-siti-nurhaliza-s-pd",
		"Renée Çelik":                 "renee-celik",
		"***":                         "tamu",
		"Keluarga Besar Alumni SMA 1": "keluarga-besar-alumni-sma-1",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueSlug(t *testing.T) {
	taken := map[string]bool{"budi": true}
	if s := UniqueSlug("Budi", taken); s != "budi-2" {
		t.Errorf("got %q", s)
	}
	if s := UniqueSlug("budi", taken); s != "budi-3" {
		t.Errorf("got %q", s)
	}
	// Nama 6 huruf tetap slug apa adanya (path dicek sebagai slug dulu, baru kode).
	if s := UniqueSlug("Andika", map[string]bool{}); s != "andika" {
		t.Errorf("got %q", s)
	}
}

func TestExtractGuestCode(t *testing.T) {
	cases := map[string]string{
		"K7P2QX":   "K7P2QX",
		" k7p2qx ": "K7P2QX",
		"https://raka-nadia.undangin.id/budi?c=K7P2QX": "K7P2QX",
		"https://raka-nadia.undangin.id/K7P2QX":        "K7P2QX",
		"https://raka-nadia.undangin.id/budi-santoso":  "",
		"K7P2Q":    "",
		"<script>": "",
	}
	for in, want := range cases {
		if got := ExtractGuestCode(in); got != want {
			t.Errorf("ExtractGuestCode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveCheckinPax(t *testing.T) {
	if n, _ := ResolveCheckinPax(0, 3); n != 3 {
		t.Errorf("default pax = %d", n)
	}
	if n, _ := ResolveCheckinPax(2, 3); n != 2 {
		t.Errorf("pax = %d", n)
	}
	if _, err := ResolveCheckinPax(4, 3); err == nil {
		t.Error("expected error for pax > invited")
	}
}

func TestGiftInputValidate(t *testing.T) {
	g, err := GiftInput{Name: " Pak  Joko ", Type: "transfer", Amount: "Rp 250.000", AccountLabel: "BCA 123"}.Validate()
	if err != nil || g.Name != "Pak Joko" || g.Amount == nil || *g.Amount != 250000 {
		t.Fatalf("got %+v err=%v", g, err)
	}
	k, err := GiftInput{Name: "Ani", Type: "kado", Amount: "5000", AccountLabel: "BCA"}.Validate()
	if err != nil || k.Amount != nil || k.AccountLabel != "" {
		t.Errorf("kado should drop amount/account: %+v", k)
	}
	if _, err := (GiftInput{Name: "A", Type: "uang"}).Validate(); err == nil {
		t.Error("expected validation error")
	}
}
