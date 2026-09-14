package domain

import (
	"net/url"
	"regexp"
	"strings"
	"time"

	"undangan/kernel/apperror"
)

const (
	CheckinOK       = "checked_in"
	CheckinAlready  = "already"
	CheckinNotFound = "not_found"
	CheckinDisabled = "disabled"
	CheckinUndo     = "undo"
)

var (
	codeRe = regexp.MustCompile(`^[A-Z0-9]{6}$`)
	pinRe  = regexp.MustCompile(`^[0-9]{6,8}$`)
)

// ExtractGuestCode menerima isi QR ("https://.../slug?c=KODE"), URL ".../KODE", atau kode yang diketik.
func ExtractGuestCode(input string) string {
	s := strings.TrimSpace(input)
	if u, err := url.Parse(s); err == nil && u.Scheme != "" {
		if c := u.Query().Get("c"); c != "" {
			s = c
		} else {
			s = u.Path[strings.LastIndex(u.Path, "/")+1:]
		}
	}
	s = strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if codeRe.MatchString(s) {
		return s
	}
	return ""
}

func IsGuestCode(s string) bool { return codeRe.MatchString(s) }

func ValidatePin(pin string) error {
	if !pinRe.MatchString(pin) {
		return apperror.Validation(map[string]string{"checkin_pin": "PIN harus 6–8 digit angka"})
	}
	return nil
}

type SettingsInput struct {
	AccessMode     *string `json:"access_mode"`
	CheckinEnabled *bool   `json:"checkin_enabled"`
	CheckinPin     *string `json:"checkin_pin"`
}

// CheckinGuest = data tamu untuk penerima tamu (tanpa no HP).
type CheckinGuest struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	GroupName    string     `json:"group_name"`
	Pax          int        `json:"pax"`
	Code         string     `json:"code"`
	CheckedInAt  *time.Time `json:"checked_in_at"`
	CheckedInPax *int       `json:"checked_in_pax"`
	CheckedInBy  string     `json:"checked_in_by"`
}

func ToCheckinGuest(g *Guest) *CheckinGuest {
	if g == nil {
		return nil
	}
	return &CheckinGuest{ID: g.ID, Name: g.Name, GroupName: g.GroupName, Pax: g.Pax, Code: g.Code, CheckedInAt: g.CheckedInAt, CheckedInPax: g.CheckedInPax, CheckedInBy: g.CheckedInBy}
}

type CheckinResult struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Guest   *CheckinGuest `json:"guest"`
}

type CheckinLog struct {
	ID           int64     `json:"id"`
	InvitationID string    `json:"-"`
	GuestID      *string   `json:"-"`
	GuestName    *string   `json:"guest_name"`
	Input        string    `json:"input"`
	Result       string    `json:"result"`
	Pax          *int      `json:"pax"`
	Station      string    `json:"station"`
	IP           string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type AttendanceSummary struct {
	InvitedGuests      int `json:"invited_guests"`
	InvitedPax         int `json:"invited_pax"`
	RSVPHadir          int `json:"rsvp_hadir"`
	RSVPPax            int `json:"rsvp_pax"`
	CheckedInGuests    int `json:"checked_in_guests"`
	CheckedInPax       int `json:"checked_in_pax"`
	NotCheckedInGuests int `json:"not_checked_in_guests"`
}

type AttendanceItem struct {
	GuestID        string     `json:"guest_id"`
	Name           string     `json:"name"`
	GroupName      string     `json:"group_name"`
	Phone          string     `json:"phone"`
	Pax            int        `json:"pax"`
	Code           string     `json:"code"`
	RSVPAttendance *string    `json:"rsvp_attendance"`
	RSVPPax        *int       `json:"rsvp_pax"`
	CheckedInAt    *time.Time `json:"checked_in_at"`
	CheckedInPax   *int       `json:"checked_in_pax"`
	CheckedInBy    string     `json:"checked_in_by"`
}

type AttendanceFilter struct {
	Query  string
	Status string // all | checked_in | not_checked_in
	Limit  int
	Offset int
}

// ResolveCheckinPax: 0 → seluruh pax undangan; selain itu 1..pax undangan.
func ResolveCheckinPax(requested, invited int) (int, error) {
	if requested <= 0 {
		return invited, nil
	}
	if requested > invited {
		return 0, apperror.Validation(map[string]string{"pax": "Jumlah yang datang melebihi jumlah yang diundang"})
	}
	return requested, nil
}
