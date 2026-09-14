package importer

import (
	"testing"

	"undangan/services/invitation/domain"
)

func TestParseCSVSemicolonWithHeaderAliases(t *testing.T) {
	data := []byte("\xEF\xBB\xBFNama Tamu;WhatsApp;Kategori;Jumlah\nBapak Joko;0812-3456-7890;Keluarga;3\n;;;\nRina;;;\n")
	rows, err := Parser{}.Parse("tamu.csv", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2 (blank line skipped)", len(rows))
	}
	if rows[0].Name != "Bapak Joko" || rows[0].Phone != "0812-3456-7890" || rows[0].Group != "Keluarga" || rows[0].Pax != "3" || rows[0].Line != 2 {
		t.Errorf("row0 = %+v", rows[0])
	}
}

func TestParseCSVWithoutHeader(t *testing.T) {
	rows, err := Parser{}.Parse("x.csv", []byte("Budi,081111111111\nAni,\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Name != "Budi" || rows[0].Phone != "081111111111" || rows[1].Line != 2 {
		t.Errorf("rows = %+v", rows)
	}
}

func TestParseXLSXTemplateRoundTrip(t *testing.T) {
	data, err := TemplateXLSX()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := Parser{}.Parse("template.xlsx", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 || rows[0].Phone != "081234567890" || rows[0].Pax != "3" {
		t.Errorf("rows = %+v", rows)
	}
}

func TestEvaluateImport(t *testing.T) {
	raw := []domain.RawRow{
		{Line: 2, Name: "Budi", Phone: "081111111111"},
		{Line: 3, Name: "budi ", Phone: "+62 811-1111-1111"}, // duplikat dalam file
		{Line: 4, Name: "Sudah Ada", Phone: ""},
		{Line: 5, Name: "", Phone: "0812"},
		{Line: 6, Name: "Ani", Pax: "dua"},
		{Line: 7, Name: "Cici"},
		{Line: 8, Name: "Dodi"}, // melebihi kuota (remaining 2)
	}
	existing := map[string]bool{domain.DedupKey("sudah ada", ""): true}
	res := domain.EvaluateImport(raw, existing, 2)

	want := []string{"ok", "duplicate", "duplicate", "error", "error", "ok", "error"}
	for i, w := range want {
		if res.Rows[i].Status != w {
			t.Errorf("row %d status = %s (%s), want %s", raw[i].Line, res.Rows[i].Status, res.Rows[i].Error, w)
		}
	}
	if res.Rows[0].Phone != "6281111111111" {
		t.Errorf("phone not normalized: %s", res.Rows[0].Phone)
	}
	if res.Summary.OK != 2 || res.Summary.Duplicate != 2 || res.Summary.Error != 3 {
		t.Errorf("summary = %+v", res.Summary)
	}
}
