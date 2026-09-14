package domain

import (
	"context"
	"time"
)

type ListFilter struct {
	UserID string // kosong = semua (admin)
	Query  string
	Status string
	Limit  int
	Offset int
}

type InvitationRepository interface {
	Create(ctx context.Context, inv *Invitation) error
	FindByID(ctx context.Context, id string) (*Invitation, error)
	// LockForUpdate mengunci baris undangan (kuota tamu, ganti subdomain) dalam transaksi.
	LockForUpdate(ctx context.Context, id string) error
	List(ctx context.Context, f ListFilter) ([]Invitation, int, error)
	CountByUser(ctx context.Context, userID string) (int, error)
	UpdateContent(ctx context.Context, id string, c Content) error
	SetTheme(ctx context.Context, id, theme string) error
	SetStatus(ctx context.Context, id string, status Status) error
	// UpdateSettings: pinHash nil = PIN tidak diubah.
	UpdateSettings(ctx context.Context, id, accessMode string, checkinEnabled bool, pinHash *string) error
	SoftDelete(ctx context.Context, id string) error

	// Subdomain
	SubdomainOwner(ctx context.Context, label string) (invitationID string, err error) // ErrNotFound bila bebas
	ReplaceSubdomain(ctx context.Context, invitationID, label string) error
	ReplaceCustomDomain(ctx context.Context, invitationID, hostname, verifyToken string) error
	MarkCustomDomainVerified(ctx context.Context, invitationID string) error
	RemoveCustomDomain(ctx context.Context, invitationID string) error

	// Halaman publik
	FindPublishedBySubdomain(ctx context.Context, label string) (*Invitation, error)
	FindPublishedByCustomDomain(ctx context.Context, hostname string) (*Invitation, error)
	FindPublishedByID(ctx context.Context, id string) (*Invitation, error)
	IsVerifiedCustomDomain(ctx context.Context, hostname string) (bool, error)

	// Lifecycle langganan
	SuspendForUser(ctx context.Context, userID string) (int, error)
	RestoreForUser(ctx context.Context, userID string) (int, error)
	PurgePersonalData(ctx context.Context, suspendedBefore time.Time) (int, error)
}

type GuestRepository interface {
	List(ctx context.Context, invitationID string, f GuestFilter) ([]Guest, int, error)
	ListAll(ctx context.Context, invitationID string) ([]Guest, error)
	Count(ctx context.Context, invitationID string) (int, error)
	Create(ctx context.Context, g *Guest) error // unique violation pada code → apperror.Conflict("code_taken")
	FindByID(ctx context.Context, invitationID, id string) (*Guest, error)
	FindByCode(ctx context.Context, invitationID, code string) (*Guest, error)
	FindBySlug(ctx context.Context, invitationID, slug string) (*Guest, error)
	// TakenSlugs = slug tamu aktif (untuk membuat slug unik).
	TakenSlugs(ctx context.Context, invitationID string) (map[string]bool, error)
	Search(ctx context.Context, invitationID, q string, limit int) ([]Guest, error)
	// CheckIn atomik: checkedIn=false bila tamu sudah check-in sebelumnya (guest berisi data check-in lama).
	CheckIn(ctx context.Context, invitationID, id string, pax int, by string) (g *Guest, checkedIn bool, err error)
	UndoCheckIn(ctx context.Context, invitationID, id string) (*Guest, error)
	Attendance(ctx context.Context, invitationID string, f AttendanceFilter) ([]AttendanceItem, int, error)
	AttendanceSummary(ctx context.Context, invitationID string) (AttendanceSummary, error)
	Update(ctx context.Context, g *Guest) error
	SoftDelete(ctx context.Context, invitationID, id string) error
	// ExistingKeys = DedupKey & kode yang sudah dipakai (untuk import).
	ExistingKeys(ctx context.Context, invitationID string) (keys map[string]bool, codes map[string]bool, err error)
	BulkInsert(ctx context.Context, guests []Guest) (int, error)
	MarkOpened(ctx context.Context, id string) error
}

type WishRepository interface {
	Create(ctx context.Context, w *Wish) error
	List(ctx context.Context, invitationID string, includeHidden bool, limit, offset int) ([]Wish, int, error)
	Stats(ctx context.Context, invitationID string, includeHidden bool) (WishStats, error)
	SetHidden(ctx context.Context, invitationID, id string, hidden bool) (*Wish, error)
	SoftDelete(ctx context.Context, invitationID, id string) error
}

type CheckinLogRepository interface {
	Insert(ctx context.Context, l *CheckinLog) error
	Recent(ctx context.Context, invitationID string, limit int) ([]CheckinLog, error)
}

type GiftRepository interface {
	Create(ctx context.Context, g *GiftConfirmation) error
	List(ctx context.Context, invitationID string, limit, offset int) ([]GiftConfirmation, int, error)
	Summary(ctx context.Context, invitationID string) (GiftSummary, error)
	FindByID(ctx context.Context, invitationID, id string) (*GiftConfirmation, error)
	SetVerified(ctx context.Context, invitationID, id string, verified bool) (*GiftConfirmation, error)
	SoftDelete(ctx context.Context, invitationID, id string) error
}

// StationTokens = token stasiun check-in (penerima tamu tanpa akun).
type StationTokens interface {
	Issue(invitationID, station, pinVersion string) (token string, expiresAt time.Time, err error)
	Verify(token string) (invitationID, station, pinVersion string, err error)
}

// ---- Port ke modul lain (diimplementasikan saat wiring) ----

// SubscriptionReader dipenuhi modul billing.
type SubscriptionReader interface {
	LimitsForUser(ctx context.Context, userID string) (status string, maxInvitations, maxGuests int, endsAt time.Time, ok bool, err error)
	DefaultGuestLimit(ctx context.Context) (int, error)
	CheckinAllowedForUser(ctx context.Context, userID string) (bool, error)
	CustomDomainAllowedForUser(ctx context.Context, userID string) (bool, error)
}

// PublicURL = alamat undangan: domain sendiri yang terverifikasi, selain itu subdomain platform.
func PublicURL(inv *Invitation, subdomainURL URLBuilder) *string {
	if inv.CustomDomain.Verified() {
		u := "https://" + inv.CustomDomain.Hostname
		return &u
	}
	if inv.Subdomain != nil {
		u := subdomainURL(*inv.Subdomain)
		return &u
	}
	return nil
}

// ThemeChecker dipenuhi modul theme.
type ThemeChecker interface {
	IsSelectable(ctx context.Context, slug string, includeInactive bool) (bool, error)
}

// GuestFileParser dipenuhi adapter importer (CSV/XLSX).
type GuestFileParser interface {
	Parse(filename string, data []byte) ([]RawRow, error)
}

// URLBuilder membentuk URL publik undangan dari subdomain.
type URLBuilder func(subdomain string) string
