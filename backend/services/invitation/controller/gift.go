package controller

import (
	"net/http"
	"strconv"
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/importer"
	"undangan/services/invitation/service"
	"undangan/kernel/apperror"
	"undangan/kernel/httpx"
)

type GiftController struct {
	svc     service.GiftService
	limiter *httpx.RateLimiter
}

func NewGiftController(svc service.GiftService) *GiftController {
	return &GiftController{svc: svc, limiter: httpx.NewRateLimiter(5, 10*time.Minute)}
}

func (c *GiftController) Register(rt *httpx.Router) {
	rt.Public("POST /api/v1/public/invitations/{id}/gifts", c.publicCreate)

	rt.Auth("GET /api/v1/invitations/{id}/gifts", c.list)
	rt.Auth("GET /api/v1/invitations/{id}/gifts/export.xlsx", c.export)
	rt.Auth("GET /api/v1/invitations/{id}/gifts/{gid}/proof", c.proof)
	rt.Auth("PATCH /api/v1/invitations/{id}/gifts/{gid}", c.setVerified)
	rt.Auth("DELETE /api/v1/invitations/{id}/gifts/{gid}", c.delete)
}

func (c *GiftController) publicCreate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	if !c.limiter.Allow(httpx.ClientIP(r) + "|" + id) {
		return apperror.TooManyRequests()
	}
	if err := httpx.ParseMultipart(w, r, domain.MaxGiftProofBytes); err != nil {
		return err
	}
	data, _, err := httpx.FormFile(r, domain.MaxGiftProofBytes)
	if err != nil {
		return apperror.Validation(map[string]string{"file": "Foto bukti wajib diunggah (maks 5 MB)"})
	}
	in := domain.GiftInput{
		Name: r.FormValue("name"), Type: r.FormValue("type"), AccountLabel: r.FormValue("account_label"),
		Amount: r.FormValue("amount"), Message: r.FormValue("message"), GuestCode: r.FormValue("guest_code"),
	}
	g, err := c.svc.PublicCreate(r.Context(), id, in, data, httpx.ClientIP(r))
	if err != nil {
		return err
	}
	httpx.Created(w, map[string]any{"id": g.ID, "created_at": g.CreatedAt})
	return nil
}

func (c *GiftController) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	items, total, sum, err := c.svc.List(r.Context(), actor(r), r.PathValue("id"), p.PerPage, p.Offset())
	if err != nil {
		return err
	}
	res := httpx.NewPaginated(items, total, p)
	httpx.OK(w, map[string]any{"items": res.Items, "total": res.Total, "page": res.Page, "per_page": res.PerPage, "summary": sum})
	return nil
}

func (c *GiftController) proof(w http.ResponseWriter, r *http.Request) error {
	f, mod, err := c.svc.Proof(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid"))
	if err != nil {
		return err
	}
	defer f.Close()
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// Content-Type dideteksi http.ServeContent dari isi file (hanya gambar yang lolos validasi saat upload).
	http.ServeContent(w, r, "bukti", mod, f)
	return nil
}

func (c *GiftController) setVerified(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		IsVerified bool `json:"is_verified"`
	}
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	g, err := c.svc.SetVerified(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid"), in.IsVerified)
	if err != nil {
		return err
	}
	httpx.OK(w, g)
	return nil
}

func (c *GiftController) delete(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.Delete(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (c *GiftController) export(w http.ResponseWriter, r *http.Request) error {
	items, _, _, err := c.svc.List(r.Context(), actor(r), r.PathValue("id"), 0, 0)
	if err != nil {
		return err
	}
	rows := make([][]string, 0, len(items))
	for _, g := range items {
		amount, guest, verified := "", "", "Belum"
		if g.Amount != nil {
			amount = strconv.FormatInt(*g.Amount, 10)
		}
		if g.GuestName != nil {
			guest = *g.GuestName
		}
		if g.IsVerified {
			verified = "Sudah"
		}
		typ := "Transfer"
		if g.Type == "kado" {
			typ = "Kado"
		}
		rows = append(rows, []string{g.CreatedAt.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04"),
			g.Name, guest, typ, g.AccountLabel, amount, g.Message, verified})
	}
	data, err := importer.ExportXLSX([]string{"Waktu", "Nama Pengirim", "Tamu Terdaftar", "Jenis", "Tujuan", "Nominal", "Pesan", "Diterima"}, rows)
	if err != nil {
		return err
	}
	attachment(w, "konfirmasi-hadiah.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
	return nil
}
