// themedev = server mandiri untuk mengembangkan tema tanpa database.
//
//	go run ./cmd/themedev            → http://localhost:8090/
//	http://localhost:8090/jawa-sogan?to=Pak+Joko
//	http://localhost:8090/jawa-sogan?guest=1   (simulasi link personal)
//	http://localhost:8090/jawa-sogan?religion=kristen&order=bride_first
//	http://localhost:8090/jawa-sogan?guest=1&checkin=1        (tiket QR check-in)
//	http://localhost:8090/jawa-sogan?access=guest_only         (form wajib kode tamu)
//	http://localhost:8090/jawa-sogan?gate=guest_only           (halaman gerbang; juga gate=not_registered&error=1)
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/renderer"
)

type wish struct {
	Name       string    `json:"name"`
	Attendance string    `json:"attendance"`
	Pax        int       `json:"pax"`
	Message    string    `json:"message"`
	CreatedAt  time.Time `json:"created_at"`
}

func main() {
	dir := os.Getenv("THEMES_DIR")
	if dir == "" {
		dir = "themes"
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8090"
	}
	r := renderer.NewRenderer(dir, true)

	var mu sync.Mutex
	wishes := []wish{
		{Name: "Dimas", Attendance: "hadir", Pax: 2, Message: "Selamat menempuh hidup baru! Semoga sakinah mawaddah warahmah.", CreatedAt: time.Now().Add(-2 * time.Hour)},
		{Name: "Keluarga Pak RT", Attendance: "ragu", Pax: 1, Message: "Barakallahu lakuma wa baraka alaikuma.", CreatedAt: time.Now().Add(-26 * time.Hour)},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /_shared/{path...}", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFileFS(w, req, r.SharedFS(), req.PathValue("path"))
	})
	mux.HandleFunc("GET /_theme/{theme}/{path...}", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFileFS(w, req, r.AssetsFS(req.PathValue("theme")), req.PathValue("path"))
	})
	mux.HandleFunc("GET /api/v1/public/invitations/{id}/wishes", func(w http.ResponseWriter, req *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		stats := map[string]int{"hadir": 0, "tidak": 0, "ragu": 0}
		for _, x := range wishes {
			stats[x.Attendance]++
		}
		items := make([]wish, len(wishes))
		for i := range wishes {
			items[i] = wishes[len(wishes)-1-i]
		}
		writeJSON(w, map[string]any{"items": items, "stats": stats, "total": len(items), "page": 1, "per_page": 20})
	})
	mux.HandleFunc("POST /api/v1/public/invitations/{id}/wishes", func(w http.ResponseWriter, req *http.Request) {
		var in wish
		if err := json.NewDecoder(req.Body).Decode(&in); err != nil || len(in.Name) < 2 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			writeJSON(w, map[string]any{"error": map[string]any{"code": "validation", "message": "Nama minimal 2 karakter"}})
			return
		}
		in.CreatedAt = time.Now()
		mu.Lock()
		wishes = append(wishes, in)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, in)
	})
	mux.HandleFunc("POST /api/v1/public/invitations/{id}/gifts", func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseMultipartForm(6 << 20); err != nil || req.FormValue("name") == "" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			writeJSON(w, map[string]any{"error": map[string]any{"code": "validation", "message": "Nama dan foto bukti wajib diisi"}})
			return
		}
		if _, _, err := req.FormFile("file"); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			writeJSON(w, map[string]any{"error": map[string]any{"code": "validation", "message": "Foto bukti wajib diunggah"}})
			return
		}
		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{"id": "demo-gift", "created_at": time.Now()})
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, req *http.Request) {
		themes, err := renderer.ScanThemes(dir)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		_ = indexTpl.Execute(w, themes)
	})
	mux.HandleFunc("GET /{theme}", func(w http.ResponseWriter, req *http.Request) {
		theme := req.PathValue("theme")
		q := req.URL.Query()
		c := renderer.DemoContent()
		if v := q.Get("religion"); v != "" {
			c.Religion = v
		}
		if v := q.Get("order"); v != "" {
			c.CoupleOrder = v
		}
		if q.Get("minimal") == "1" { // uji section kosong
			c.LoveStory, c.Gift, c.LiveStream = nil, domain.Gift{}, domain.LiveStream{}
			c.Closing.Family = nil
			c.Events = c.Events[:1]
		}
		var guest *renderer.GuestView
		if q.Get("guest") == "1" {
			guest = &renderer.GuestView{Name: "Bapak Joko Santoso & Keluarga", Pax: 3, Code: "K7P2QX", Slug: "bapak-joko-santoso-keluarga",
				Link: "http://localhost" + addr + "/" + theme + "/bapak-joko-santoso-keluarga"}
		}
		inv := &renderer.Invitation{ID: "demo", Theme: theme}
		data := renderer.BuildPageData(inv, c, guest, renderer.GuestNameFromQuery(q.Get("to")), q.Get("preview") == "1", "http://localhost"+addr+"/"+theme)
		data = data.WithAccess(q.Get("access"), q.Get("checkin") == "1")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if gate := q.Get("gate"); gate != "" {
			g := renderer.GateData{PageData: data, Reason: gate, Attempted: "budi-tidak-ada"}
			if q.Get("error") == "1" {
				g.Error = "Kode undangan tidak valid"
			}
			if err := r.RenderGate(w, theme, g); err != nil {
				log.Printf("gate %s: %v", theme, err)
				http.Error(w, err.Error(), 500)
			}
			return
		}
		if err := r.Render(w, theme, data); err != nil {
			log.Printf("render %s: %v", theme, err)
			http.Error(w, err.Error(), 500)
		}
	})

	fmt.Printf("themedev: http://localhost%s/\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

var indexTpl = template.Must(template.New("i").Parse(`<!doctype html><meta charset=utf-8><title>Tema</title>
<body style="font-family:system-ui;max-width:48rem;margin:2rem auto;padding:0 1rem">
<h1>Katalog tema</h1><ul>{{range .}}<li><a href="/{{.Slug}}?to=Pak+Joko">{{.Name}}</a> — {{.Description}}
 · <a href="/{{.Slug}}?guest=1">personal</a> · <a href="/{{.Slug}}?minimal=1">minimal</a> · <a href="/{{.Slug}}?guest=1&checkin=1">tiket</a> · <a href="/{{.Slug}}?gate=guest_only">gerbang</a> · <a href="/{{.Slug}}?gate=not_registered">tidak terdaftar</a></li>{{end}}</ul>`))
