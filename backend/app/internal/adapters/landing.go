// Package adapter menghubungkan service modul lain ke port landing/domain.
// Hanya bergantung ke domain modul penyedia (bukan service/repository), jadi aman diimpor di cmd/server:
//
//	landingsvc.New(
//	adapters.LandingThemes{Svc: themeService},          // theme/service.Service
//	adapters.LandingPlans{Svc: planService},            // billing/service.PlanService
//	adapters.LandingContact{Svc: settingsService},      // billing/service.SettingsService
//		landingsvc.Config{AppName: cfg.AppName, PortalURL: cfg.PortalURL, BaseDomain: cfg.BaseDomain},
//	)
package adapters

import (
	"context"

	billingdomain "undangan/services/billing/domain"
	"undangan/services/landing/domain"
	themedomain "undangan/services/theme/domain"
)

// ThemeSource = subset theme/service.Service.
type ThemeSource interface {
	List(ctx context.Context, onlyActive bool) ([]themedomain.Theme, error)
}

// PlanSource = subset billing/service.PlanService.
type PlanSource interface {
	ListActive(ctx context.Context) ([]billingdomain.Plan, error)
}

// PaymentSettingsSource = subset billing/service.SettingsService.
type PaymentSettingsSource interface {
	Payment(ctx context.Context) (*billingdomain.PaymentSettings, error)
}

type LandingThemes struct{ Svc ThemeSource }

func (a LandingThemes) ListThemes(ctx context.Context) ([]domain.Theme, error) {
	items, err := a.Svc.List(ctx, true)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Theme, 0, len(items))
	for _, t := range items {
		if !t.IsActive {
			continue
		}
		out = append(out, domain.Theme{
			Slug: t.Slug, Name: t.Name, Description: t.Description, Category: t.Category,
			Thumbnail: t.Thumbnail, PreviewURL: t.PreviewURL, Colors: t.Colors,
		})
	}
	return out, nil
}

type LandingPlans struct{ Svc PlanSource }

func (a LandingPlans) ListPlans(ctx context.Context) ([]domain.Plan, error) {
	items, err := a.Svc.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Plan, 0, len(items))
	for _, p := range items {
		out = append(out, domain.Plan{
			ID: p.ID, Name: p.Name, Description: p.Description, Price: p.Price,
			DurationDays: p.DurationDays, GraceDays: p.GraceDays, MaxInvitations: p.MaxInvitations, MaxGuests: p.MaxGuests,
			AllowCustomDomain: p.AllowCustomDomain, AllowCheckin: p.AllowCheckin,
		})
	}
	return out, nil
}

type LandingContact struct{ Svc PaymentSettingsSource }

func (a LandingContact) Contact(ctx context.Context) (domain.Contact, error) {
	ps, err := a.Svc.Payment(ctx)
	if err != nil || ps == nil {
		return domain.Contact{}, err
	}
	return domain.Contact{AdminWhatsApp: ps.AdminWhatsApp, MerchantName: ps.MerchantName}, nil
}

var (
	_ domain.ThemeLister     = LandingThemes{}
	_ domain.PlanLister      = LandingPlans{}
	_ domain.ContactProvider = LandingContact{}
)
