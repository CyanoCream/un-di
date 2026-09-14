// Package controller (invitation) = HTTP handler API undangan, tamu, ucapan, dan halaman publik.
package controller

import (
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/service"
)

// Bentuk JSON mengikuti packages/shared/src/types.ts (InvitationSummary & Invitation).
type summaryDTO struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	UserName    string        `json:"user_name"`
	UserEmail   string        `json:"user_email"`
	Title       string        `json:"title"`
	Theme       *string       `json:"theme"`
	ThemeLocked bool          `json:"theme_locked"`
	Subdomain   *string       `json:"subdomain"`
	URL         *string       `json:"url"`
	Status      domain.Status `json:"status"`
	EventDate   *string       `json:"event_date"`
	GuestCount  int           `json:"guest_count"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type quotaDTO struct {
	MaxGuests  int `json:"max_guests"`
	UsedGuests int `json:"used_guests"`
}

type settingsDTO struct {
	AccessMode            string `json:"access_mode"`
	CheckinEnabled        bool   `json:"checkin_enabled"`
	CheckinPinSet         bool   `json:"checkin_pin_set"`
	CheckinAvailable      bool   `json:"checkin_available"`
	CheckinStationPath    string `json:"checkin_station_path"`
	CustomDomainAvailable bool   `json:"custom_domain_available"`
}

type customDomainDTO struct {
	Hostname   string             `json:"hostname"`
	Verified   bool               `json:"verified"`
	VerifiedAt *time.Time         `json:"verified_at"`
	DNS        []domain.DNSRecord `json:"dns"`
}

type detailDTO struct {
	summaryDTO
	Content            domain.Content   `json:"content"`
	Settings           settingsDTO      `json:"settings"`
	CustomDomain       *customDomainDTO `json:"custom_domain"`
	Quota              quotaDTO         `json:"quota"`
	SubscriptionStatus *string          `json:"subscription_status"`
}

func toSummary(inv *domain.Invitation, url domain.URLBuilder) summaryDTO {
	d := summaryDTO{
		ID: inv.ID, UserID: inv.UserID, UserName: inv.OwnerName, UserEmail: inv.OwnerEmail,
		Title: inv.Title(), Theme: inv.Theme, ThemeLocked: inv.ThemeLocked(), Subdomain: inv.Subdomain,
		Status: inv.Status, EventDate: inv.EventDate(), GuestCount: inv.GuestCount,
		CreatedAt: inv.CreatedAt, UpdatedAt: inv.UpdatedAt,
	}
	d.URL = domain.PublicURL(inv, url)
	return d
}

func toSummaries(items []domain.Invitation, url domain.URLBuilder) []summaryDTO {
	out := make([]summaryDTO, 0, len(items))
	for i := range items {
		out = append(out, toSummary(&items[i], url))
	}
	return out
}

func toDetail(d *service.Details, url domain.URLBuilder, dcfg domain.DomainConfig) detailDTO {
	var cd *customDomainDTO
	if c := d.Invitation.CustomDomain; c != nil {
		cd = &customDomainDTO{Hostname: c.Hostname, Verified: c.Verified(), VerifiedAt: c.VerifiedAt, DNS: domain.DNSInstructions(c, dcfg)}
	}
	return detailDTO{
		CustomDomain: cd,
		summaryDTO:   toSummary(d.Invitation, url),
		Content:      d.Invitation.Content,
		Settings: settingsDTO{
			AccessMode: d.Invitation.AccessMode, CheckinEnabled: d.Invitation.CheckinEnabled,
			CheckinPinSet: d.Invitation.CheckinPinHash != "", CheckinAvailable: d.CheckinAvailable,
			CheckinStationPath: "/checkin/" + d.Invitation.ID, CustomDomainAvailable: d.CustomDomainAvailable,
		},
		Quota:              quotaDTO{MaxGuests: d.MaxGuests, UsedGuests: d.Invitation.GuestCount},
		SubscriptionStatus: d.SubscriptionStatus,
	}
}
