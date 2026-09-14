package domain

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const maxSlugLen = 60

// Slugify: "Bapak Joko & Keluarga" → "bapak-joko-keluarga"; huruf beraksen → huruf dasar. Kosong → "tamu".
func Slugify(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range norm.NFD.String(strings.ToLower(name)) {
		if b.Len() >= maxSlugLen {
			break
		}
		switch {
		case unicode.Is(unicode.Mn, r): // tanda diakritik dibuang
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			dash = false
		default:
			if !dash && b.Len() > 0 {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "tamu"
	}
	return s
}

// UniqueSlug menambahkan -2, -3, ... bila slug sudah dipakai; taken diperbarui.
func UniqueSlug(name string, taken map[string]bool) string {
	base := Slugify(name)
	s := base
	for i := 2; taken[s]; i++ {
		s = base + "-" + strconv.Itoa(i)
	}
	taken[s] = true
	return s
}
