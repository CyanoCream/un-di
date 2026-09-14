package domain

import (
	"net/url"
	"strconv"
	"strings"
	"unicode"
)

// categoryOrder = urutan kanonik kategori beserta label Bahasa Indonesia.
var categoryOrder = []Category{
	{Key: "adat", Label: "Adat Nusantara"},
	{Key: "islami", Label: "Islami"},
	{Key: "floral", Label: "Floral"},
	{Key: "modern", Label: "Modern & Minimalis"},
	{Key: "elegan", Label: "Elegan & Mewah"},
	{Key: "rustic", Label: "Rustic & Boho"},
	{Key: "pastel", Label: "Pastel"},
	{Key: "retro", Label: "Retro"},
}

// CategoryLabel mengembalikan label Indonesia; kategori tak dikenal → "Lainnya".
func CategoryLabel(key string) string {
	for _, c := range categoryOrder {
		if c.Key == key {
			return c.Label
		}
	}
	return "Lainnya"
}

// ValidCategory = kategori termasuk daftar kanonik.
func ValidCategory(key string) bool {
	for _, c := range categoryOrder {
		if c.Key == key {
			return true
		}
	}
	return false
}

// FormatRupiah: 175000 → "Rp175.000" (gaya PUEBI tanpa spasi).
func FormatRupiah(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	if neg {
		return "-Rp" + b.String()
	}
	return "Rp" + b.String()
}

// FormatNumber: 5000 → "5.000".
func FormatNumber(n int) string { return strings.TrimPrefix(FormatRupiah(int64(n)), "Rp") }

// FormatDuration: 30 → "30 hari", 365 → "1 tahun".
func FormatDuration(days int) string {
	if days >= 365 && days%365 == 0 {
		return strconv.Itoa(days/365) + " tahun"
	}
	return strconv.Itoa(days) + " hari"
}

// WhatsAppLink membentuk wa.me dengan pesan terisi; nomor harus 62xx.
func WhatsAppLink(number, text string) string {
	number = digitsOnly(number)
	if !strings.HasPrefix(number, "62") || len(number) < 9 {
		return ""
	}
	u := "https://wa.me/" + number
	if text != "" {
		u += "?text=" + url.QueryEscape(text)
	}
	return u
}

// WhatsAppDisplay: 6281234567890 → "+62 812-3456-7890".
func WhatsAppDisplay(number string) string {
	number = digitsOnly(number)
	if !strings.HasPrefix(number, "62") || len(number) < 9 {
		return ""
	}
	rest := number[2:]
	var parts []string
	for len(rest) > 4 {
		n := 4
		if len(parts) == 0 {
			n = 3
		}
		parts = append(parts, rest[:n])
		rest = rest[n:]
	}
	if rest != "" {
		parts = append(parts, rest)
	}
	return "+62 " + strings.Join(parts, "-")
}

// Initials: "Jawa Sogan" → "JS".
func Initials(name string) string {
	var out []rune
	for _, w := range strings.Fields(name) {
		for _, r := range w {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				out = append(out, unicode.ToUpper(r))
				break
			}
		}
		if len(out) == 2 {
			break
		}
	}
	return string(out)
}

// JoinID: ["A"] → "A"; ["A","B"] → "A dan B"; ["A","B","C"] → "A, B, dan C".
func JoinID(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " dan " + items[1]
	}
	return strings.Join(items[:len(items)-1], ", ") + ", dan " + items[len(items)-1]
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
