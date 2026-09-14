package renderer

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"
)

var themeSlugRe = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

func ValidThemeSlug(s string) bool { return themeSlugRe.MatchString(s) && !strings.HasPrefix(s, "_") }

// Renderer memuat themes/_shared/*.html + themes/<slug>/index.html.
// Tema boleh override partial _shared dengan {{define}} bernama sama di index.html.
type Renderer struct {
	dir   string
	dev   bool
	mu    sync.RWMutex
	cache map[string]*template.Template
}

func NewRenderer(dir string, dev bool) *Renderer {
	return &Renderer{dir: dir, dev: dev, cache: map[string]*template.Template{}}
}

var funcs = template.FuncMap{
	"upper": strings.ToUpper,
	"initial": func(s string) string {
		for _, r := range strings.TrimSpace(s) {
			return string(unicode.ToUpper(r))
		}
		return ""
	},
	// paras memecah teks multi-baris jadi paragraf (tetap di-escape oleh template).
	"paras": func(s string) []string {
		var out []string
		for _, p := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	},
	"seq": func(n int) []int {
		out := make([]int, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, i)
		}
		return out
	},
	"add":          func(a, b int) int { return a + b },
	"youtubeEmbed": youtubeEmbed,
}

func (r *Renderer) Render(w io.Writer, theme string, data PageData) error {
	t, err := r.load(theme)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, "index.html", data)
}

// RenderGate merender halaman gerbang: themes/_shared/gate.html (tema boleh override {{define "gate.html"}}).
func (r *Renderer) RenderGate(w io.Writer, theme string, data GateData) error {
	t, err := r.load(theme)
	if err != nil {
		return err
	}
	if t.Lookup("gate.html") == nil {
		return fmt.Errorf("template gate.html tidak ditemukan")
	}
	return t.ExecuteTemplate(w, "gate.html", data)
}

func (r *Renderer) load(theme string) (*template.Template, error) {
	if !ValidThemeSlug(theme) {
		return nil, fmt.Errorf("invalid theme slug %q", theme)
	}
	if !r.dev {
		r.mu.RLock()
		t, ok := r.cache[theme]
		r.mu.RUnlock()
		if ok {
			return t, nil
		}
	}

	t := template.New("index.html").Funcs(funcs)
	shared, err := filepath.Glob(filepath.Join(r.dir, "_shared", "*.html"))
	if err != nil {
		return nil, err
	}
	if len(shared) > 0 {
		if t, err = t.ParseFiles(shared...); err != nil {
			return nil, err
		}
	}
	if t, err = t.ParseFiles(filepath.Join(r.dir, theme, "index.html")); err != nil {
		return nil, err
	}

	if !r.dev {
		r.mu.Lock()
		r.cache[theme] = t
		r.mu.Unlock()
	}
	return t, nil
}

// AssetsFS = file statis tema di themes/<slug>/assets.
func (r *Renderer) AssetsFS(theme string) fs.FS {
	return os.DirFS(filepath.Join(r.dir, theme, "assets"))
}

// SharedFS = runtime.js, base.css, ikon bersama di themes/_shared.
func (r *Renderer) SharedFS() fs.FS {
	return os.DirFS(filepath.Join(r.dir, "_shared"))
}

var ytRe = regexp.MustCompile(`(?:youtu\.be/|youtube\.com/(?:watch\?v=|embed/|shorts/|live/))([A-Za-z0-9_-]{11})`)

func youtubeEmbed(u string) string {
	if m := ytRe.FindStringSubmatch(u); m != nil {
		return "https://www.youtube-nocookie.com/embed/" + m[1]
	}
	return ""
}
