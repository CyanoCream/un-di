package controller

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"undangan/services/invitation/domain"
	"undangan/services/invitation/importer"
	"undangan/services/invitation/service"
	"undangan/kernel/httpx"
)

type GuestController struct {
	svc service.GuestService
}

func NewGuestController(svc service.GuestService) *GuestController {
	return &GuestController{svc: svc}
}

func (c *GuestController) Register(rt *httpx.Router) {
	rt.Auth("GET /api/v1/invitations/{id}/guests", c.list)
	rt.Auth("POST /api/v1/invitations/{id}/guests", c.create)
	rt.Auth("PATCH /api/v1/invitations/{id}/guests/{gid}", c.update)
	rt.Auth("DELETE /api/v1/invitations/{id}/guests/{gid}", c.delete)
	rt.Auth("POST /api/v1/invitations/{id}/guests/import", c.importFile)
	rt.Auth("GET /api/v1/invitations/{id}/guests/template.csv", c.templateCSV)
	rt.Auth("GET /api/v1/invitations/{id}/guests/template.xlsx", c.templateXLSX)
	rt.Auth("GET /api/v1/invitations/{id}/guests/export.csv", c.exportCSV)
	rt.Auth("GET /api/v1/invitations/{id}/guests/export.xlsx", c.exportXLSX)
	rt.Auth("POST /api/v1/invitations/{id}/guests/{gid}/checkin", c.checkIn)
	rt.Auth("DELETE /api/v1/invitations/{id}/guests/{gid}/checkin", c.undoCheckIn)
	rt.Auth("GET /api/v1/invitations/{id}/attendance", c.attendance)
	rt.Auth("GET /api/v1/invitations/{id}/attendance/export.xlsx", c.attendanceExport)
}

func (c *GuestController) list(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, err := c.svc.List(r.Context(), actor(r), r.PathValue("id"), domain.GuestFilter{
		Query: strings.TrimSpace(q.Get("q")), Group: q.Get("group"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	httpx.OK(w, httpx.NewPaginated(items, total, p))
	return nil
}

func (c *GuestController) create(w http.ResponseWriter, r *http.Request) error {
	var in domain.GuestInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	g, err := c.svc.Create(r.Context(), actor(r), r.PathValue("id"), in)
	if err != nil {
		return err
	}
	httpx.Created(w, g)
	return nil
}

func (c *GuestController) update(w http.ResponseWriter, r *http.Request) error {
	var in domain.GuestInput
	if err := httpx.Decode(r, &in); err != nil {
		return err
	}
	g, err := c.svc.Update(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid"), in)
	if err != nil {
		return err
	}
	httpx.OK(w, g)
	return nil
}

func (c *GuestController) delete(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.Delete(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (c *GuestController) importFile(w http.ResponseWriter, r *http.Request) error {
	if err := httpx.ParseMultipart(w, r, domain.MaxImportBytes); err != nil {
		return err
	}
	data, hdr, err := httpx.FormFile(r, domain.MaxImportBytes)
	if err != nil {
		return err
	}
	dry, _ := strconv.ParseBool(r.URL.Query().Get("dry_run"))
	res, err := c.svc.Import(r.Context(), actor(r), r.PathValue("id"), hdr.Filename, data, dry)
	if err != nil {
		return err
	}
	httpx.OK(w, res)
	return nil
}

func attachment(w http.ResponseWriter, filename, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(data)
}

func (c *GuestController) templateCSV(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.CanAccess(r.Context(), actor(r), r.PathValue("id")); err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF") // BOM supaya Excel membaca UTF-8
	cw := csv.NewWriter(&buf)
	_ = cw.WriteAll([][]string{
		{"nama", "no_hp", "grup", "jumlah_tamu"},
		{"Bapak Joko Santoso & Keluarga", "081234567890", "Keluarga", "3"},
		{"Rina Amelia", "085712345678", "Teman Kantor", "1"},
	})
	attachment(w, "template-tamu.csv", "text/csv; charset=utf-8", buf.Bytes())
	return nil
}

func (c *GuestController) templateXLSX(w http.ResponseWriter, r *http.Request) error {
	if err := c.svc.CanAccess(r.Context(), actor(r), r.PathValue("id")); err != nil {
		return err
	}
	data, err := importer.TemplateXLSX()
	if err != nil {
		return err
	}
	attachment(w, "template-tamu.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
	return nil
}

func (c *GuestController) checkIn(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Pax int `json:"pax"`
	}
	if r.ContentLength > 0 {
		if err := httpx.Decode(r, &in); err != nil {
			return err
		}
	}
	g, err := c.svc.CheckIn(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid"), in.Pax)
	if err != nil {
		return err
	}
	httpx.OK(w, g)
	return nil
}

func (c *GuestController) undoCheckIn(w http.ResponseWriter, r *http.Request) error {
	g, err := c.svc.UndoCheckIn(r.Context(), actor(r), r.PathValue("id"), r.PathValue("gid"))
	if err != nil {
		return err
	}
	httpx.OK(w, g)
	return nil
}

func (c *GuestController) attendance(w http.ResponseWriter, r *http.Request) error {
	p := httpx.PageFrom(r)
	q := r.URL.Query()
	items, total, sum, err := c.svc.Attendance(r.Context(), actor(r), r.PathValue("id"), domain.AttendanceFilter{
		Query: strings.TrimSpace(q.Get("q")), Status: q.Get("status"), Limit: p.PerPage, Offset: p.Offset(),
	})
	if err != nil {
		return err
	}
	res := httpx.NewPaginated(items, total, p)
	httpx.OK(w, map[string]any{"items": res.Items, "total": res.Total, "page": res.Page, "per_page": res.PerPage, "summary": sum})
	return nil
}

func (c *GuestController) attendanceExport(w http.ResponseWriter, r *http.Request) error {
	items, _, sum, err := c.svc.Attendance(r.Context(), actor(r), r.PathValue("id"), domain.AttendanceFilter{Status: "all"})
	if err != nil {
		return err
	}
	wib := time.FixedZone("WIB", 7*3600)
	rsvpLabel := map[string]string{"hadir": "Hadir", "tidak": "Tidak hadir", "ragu": "Masih ragu"}
	rows := make([][]string, 0, len(items)+3)
	for _, it := range items {
		rsvp, checkin, pax := "-", "Belum datang", ""
		if it.RSVPAttendance != nil {
			rsvp = rsvpLabel[*it.RSVPAttendance]
		}
		if it.CheckedInAt != nil {
			checkin = it.CheckedInAt.In(wib).Format("2006-01-02 15:04")
		}
		if it.CheckedInPax != nil {
			pax = strconv.Itoa(*it.CheckedInPax)
		}
		rows = append(rows, []string{it.Name, it.GroupName, it.Phone, strconv.Itoa(it.Pax), it.Code, rsvp, checkin, pax, it.CheckedInBy})
	}
	rows = append(rows, []string{}, []string{"Ringkasan", "", "", strconv.Itoa(sum.InvitedPax), "", "RSVP hadir: " + strconv.Itoa(sum.RSVPHadir),
		fmt.Sprintf("Datang: %d dari %d tamu", sum.CheckedInGuests, sum.InvitedGuests), strconv.Itoa(sum.CheckedInPax), ""})
	data, err := importer.ExportXLSX([]string{"Nama", "Grup", "No HP", "Diundang (orang)", "Kode", "RSVP", "Check-in", "Datang (orang)", "Penerima"}, rows)
	if err != nil {
		return err
	}
	attachment(w, "laporan-kehadiran.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
	return nil
}

var exportHeader = []string{"Nama", "No HP", "Grup", "Jumlah Tamu", "Kode", "Link Undangan", "Dibuka", "Check-in"}

func exportRows(guests []domain.Guest) [][]string {
	ts := func(t *time.Time) string {
		if t == nil {
			return ""
		}
		return t.In(time.FixedZone("WIB", 7*3600)).Format("2006-01-02 15:04")
	}
	rows := make([][]string, 0, len(guests))
	for _, g := range guests {
		link := ""
		if g.Link != nil {
			link = *g.Link
		}
		rows = append(rows, []string{g.Name, g.Phone, g.GroupName, strconv.Itoa(g.Pax), g.Code, link, ts(g.OpenedAt), ts(g.CheckedInAt)})
	}
	return rows
}

func exportName(inv *domain.Invitation, ext string) string {
	name := "tamu"
	if inv.Subdomain != nil {
		name = "tamu-" + *inv.Subdomain
	}
	return name + "." + ext
}

func (c *GuestController) exportCSV(w http.ResponseWriter, r *http.Request) error {
	inv, guests, err := c.svc.Export(r.Context(), actor(r), r.PathValue("id"))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	buf.WriteString("\xEF\xBB\xBF")
	cw := csv.NewWriter(&buf)
	_ = cw.Write(exportHeader)
	_ = cw.WriteAll(exportRows(guests))
	attachment(w, exportName(inv, "csv"), "text/csv; charset=utf-8", buf.Bytes())
	return nil
}

func (c *GuestController) exportXLSX(w http.ResponseWriter, r *http.Request) error {
	inv, guests, err := c.svc.Export(r.Context(), actor(r), r.PathValue("id"))
	if err != nil {
		return err
	}
	data, err := importer.ExportXLSX(exportHeader, exportRows(guests))
	if err != nil {
		return err
	}
	attachment(w, exportName(inv, "xlsx"), "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
	return nil
}
