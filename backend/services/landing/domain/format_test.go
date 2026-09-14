package domain

import "testing"

func TestFormatters(t *testing.T) {
	for in, want := range map[int64]string{0: "Rp0", 999: "Rp999", 1000: "Rp1.000", 175000: "Rp175.000", 1250000: "Rp1.250.000", -5000: "-Rp5.000"} {
		if got := FormatRupiah(in); got != want {
			t.Errorf("FormatRupiah(%d)=%q want %q", in, got, want)
		}
	}
	if got := FormatDuration(30); got != "30 hari" {
		t.Errorf("FormatDuration(30)=%q", got)
	}
	if got := FormatDuration(730); got != "2 tahun" {
		t.Errorf("FormatDuration(730)=%q", got)
	}
	if got := JoinID([]string{"A", "B", "C"}); got != "A, B, dan C" {
		t.Errorf("JoinID=%q", got)
	}
	if got := Initials("Surat Segel Lilin"); got != "SS" {
		t.Errorf("Initials=%q", got)
	}
	if got := WhatsAppLink("+62 812-3456-7890", "Halo & salam"); got != "https://wa.me/6281234567890?text=Halo+%26+salam" {
		t.Errorf("WhatsAppLink=%q", got)
	}
	if got := WhatsAppLink("", "x"); got != "" {
		t.Errorf("WhatsAppLink empty=%q", got)
	}
}

func TestBuildPlansHighlightNeedsTwoPaidPlans(t *testing.T) {
	one := BuildPlans([]Plan{{Name: "A", Price: 100, MaxGuests: 10, DurationDays: 30}})
	if one[0].Highlight {
		t.Fatal("single plan must not be highlighted")
	}
	free := BuildPlans([]Plan{{Name: "Free", Price: 0, MaxGuests: 10, DurationDays: 7}, {Name: "B", Price: 50, MaxGuests: 10, DurationDays: 30}})
	if free[0].PriceLabel != "Gratis" || free[0].Highlight || free[1].Highlight {
		t.Fatalf("free plan handling: %+v", free)
	}
}

func TestBuildPlansHidesUnavailableExtras(t *testing.T) {
	v := BuildPlans([]Plan{{Name: "A", Price: 100, MaxGuests: 10, DurationDays: 30}})
	for _, it := range v[0].Items {
		if !it.Included {
			t.Fatalf("no plan offers %q, it must not be listed as excluded", it.Text)
		}
	}
}
