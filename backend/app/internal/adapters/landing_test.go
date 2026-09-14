package adapters

import (
	"context"
	"testing"

	billingdomain "undangan/services/billing/domain"
	themedomain "undangan/services/theme/domain"
)

type themeSrc []themedomain.Theme

func (s themeSrc) List(_ context.Context, onlyActive bool) ([]themedomain.Theme, error) {
	return s, nil
}

type planSrc []billingdomain.Plan

func (s planSrc) ListActive(context.Context) ([]billingdomain.Plan, error) { return s, nil }

type paySrc struct {
	ps *billingdomain.PaymentSettings
}

func (s paySrc) Payment(context.Context) (*billingdomain.PaymentSettings, error) { return s.ps, nil }

func TestAdapters(t *testing.T) {
	ctx := context.Background()
	th, err := LandingThemes{Svc: themeSrc{
		{Slug: "a", Name: "A", Category: "adat", Thumbnail: "/_theme/a/thumb.webp", PreviewURL: "/_preview/a", Colors: []string{"#fff"}, IsActive: true},
		{Slug: "b", Name: "B", IsActive: false},
	}}.ListThemes(ctx)
	if err != nil || len(th) != 1 || th[0].Thumbnail != "/_theme/a/thumb.webp" || th[0].Category != "adat" {
		t.Fatalf("themes: %+v %v", th, err)
	}
	pl, err := LandingPlans{Svc: planSrc{{ID: "p", Name: "P", Price: 5, MaxGuests: 10, AllowCheckin: true}}}.ListPlans(ctx)
	if err != nil || len(pl) != 1 || !pl[0].AllowCheckin || pl[0].Price != 5 {
		t.Fatalf("plans: %+v %v", pl, err)
	}
	ct, err := LandingContact{Svc: paySrc{&billingdomain.PaymentSettings{AdminWhatsApp: "628123", MerchantName: "M"}}}.Contact(ctx)
	if err != nil || ct.AdminWhatsApp != "628123" || ct.MerchantName != "M" {
		t.Fatalf("contact: %+v %v", ct, err)
	}
	if ct, err := (LandingContact{Svc: paySrc{}}).Contact(ctx); err != nil || ct.AdminWhatsApp != "" {
		t.Fatalf("nil settings: %+v %v", ct, err)
	}
}
