package renderer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// ThemeMeta dibaca dari themes/<slug>/theme.json.
type ThemeMeta struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Colors      []string `json:"colors"`
	Fonts       []string `json:"fonts"`
	Category    string   `json:"category"` // adat | islami | floral | modern | elegan | rustic | pastel | retro
	Thumbnail   string   `json:"-"`        // /_theme/<slug>/thumb.webp bila file ada
}

// ScanThemes membaca semua folder tema yang punya index.html + theme.json.
func ScanThemes(dir string) ([]ThemeMeta, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []ThemeMeta
	for _, e := range entries {
		if !e.IsDir() || !ValidThemeSlug(e.Name()) {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "index.html")); err != nil {
			continue
		}
		meta := ThemeMeta{Slug: e.Name(), Name: e.Name()}
		if b, err := os.ReadFile(filepath.Join(dir, e.Name(), "theme.json")); err == nil {
			_ = json.Unmarshal(b, &meta)
			meta.Slug = e.Name()
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "assets", "thumb.webp")); err == nil {
			meta.Thumbnail = "/_theme/" + e.Name() + "/thumb.webp"
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}
