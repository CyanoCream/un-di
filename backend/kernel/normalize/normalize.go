// Package normalize = normalisasi input yang dipakai lintas service.
package normalize

import "strings"

func Email(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// Phone: "0812-3456 789" / "+62812…" / "812…" → "62812…"; nilai tidak wajar → "".
func Phone(p string) string {
	var b strings.Builder
	for _, r := range p {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	s := b.String()
	switch {
	case s == "":
		return ""
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	case strings.HasPrefix(s, "8"):
		s = "62" + s
	}
	if len(s) < 9 || len(s) > 15 {
		return ""
	}
	return s
}
