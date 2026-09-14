// Package importer = adapter pembaca file daftar tamu (CSV / XLSX) → domain.RawRow.
package importer

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"

	"undangan/services/invitation/domain"
	"undangan/kernel/apperror"
)

type Parser struct{}

var _ domain.GuestFileParser = Parser{}

func (Parser) Parse(filename string, data []byte) ([]domain.RawRow, error) {
	var records [][]string
	var err error
	// XLSX = arsip ZIP ("PK\x03\x04"); selain itu diperlakukan sebagai CSV.
	if bytes.HasPrefix(data, []byte("PK\x03\x04")) {
		records, err = readXLSX(data)
	} else {
		if strings.HasSuffix(strings.ToLower(filename), ".xls") {
			return nil, apperror.BadRequest("Format .xls lama tidak didukung. Simpan ulang sebagai .xlsx atau .csv")
		}
		records, err = readCSV(data)
	}
	if err != nil {
		return nil, err
	}
	return mapRows(records)
}

func readXLSX(data []byte) ([][]string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, apperror.BadRequest("File Excel tidak bisa dibaca")
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, apperror.BadRequest("File Excel kosong")
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, apperror.BadRequest("Sheet pertama tidak bisa dibaca")
	}
	return rows, nil
}

func readCSV(data []byte) ([][]string, error) {
	data = bytes.TrimPrefix(data, []byte("\xEF\xBB\xBF")) // BOM dari Excel
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = detectDelimiter(data)
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	var out [][]string
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, apperror.BadRequest(fmt.Sprintf("CSV tidak valid: %v", err))
		}
		out = append(out, rec)
	}
	return out, nil
}

// detectDelimiter: Excel berbahasa Indonesia menyimpan CSV dengan ';'.
func detectDelimiter(data []byte) rune {
	line := data
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		line = data[:i]
	}
	best, bestN := ',', bytes.Count(line, []byte{','})
	for _, d := range []rune{';', '\t'} {
		if n := bytes.Count(line, []byte(string(d))); n > bestN {
			best, bestN = d, n
		}
	}
	return best
}

func mapRows(records [][]string) ([]domain.RawRow, error) {
	if len(records) == 0 {
		return nil, apperror.BadRequest("File tidak berisi data")
	}
	// Kolom default tanpa header: nama, no HP, grup, jumlah tamu.
	idx := map[string]int{"name": 0, "phone": 1, "group": 2, "pax": 3}
	start := 0
	found := map[string]int{}
	for i, h := range records[0] {
		if f := domain.HeaderField(h); f != "" {
			if _, dup := found[f]; !dup {
				found[f] = i
			}
		}
	}
	if _, ok := found["name"]; ok {
		idx = map[string]int{"name": -1, "phone": -1, "group": -1, "pax": -1}
		for k, v := range found {
			idx[k] = v
		}
		start = 1
	}

	cell := func(rec []string, field string) string {
		i := idx[field]
		if i < 0 || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	var rows []domain.RawRow
	for i := start; i < len(records); i++ {
		rec := records[i]
		row := domain.RawRow{Line: i + 1, Name: cell(rec, "name"), Phone: cell(rec, "phone"), Group: cell(rec, "group"), Pax: cell(rec, "pax")}
		if row.Name == "" && row.Phone == "" && row.Group == "" && row.Pax == "" {
			continue // baris kosong
		}
		rows = append(rows, row)
		if len(rows) > domain.MaxImportRows {
			return nil, apperror.BadRequest(fmt.Sprintf("Maksimal %d baris per file", domain.MaxImportRows))
		}
	}
	if len(rows) == 0 {
		return nil, apperror.BadRequest("Tidak ada baris tamu yang terbaca")
	}
	return rows, nil
}

// TemplateXLSX = file contoh untuk diunduh customer.
func TemplateXLSX() ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Daftar Tamu"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	rows := [][]any{
		{"nama", "no_hp", "grup", "jumlah_tamu"},
		{"Bapak Joko Santoso & Keluarga", "081234567890", "Keluarga", 3},
		{"Rina Amelia", "085712345678", "Teman Kantor", 1},
		{"Keluarga Besar Alumni SMA 1", "", "Alumni", 2},
	}
	for i, r := range rows {
		if err := f.SetSheetRow(sheet, fmt.Sprintf("A%d", i+1), &r); err != nil {
			return nil, err
		}
	}
	// Kolom no_hp sebagai teks supaya angka 0 di depan tidak hilang.
	textStyle, _ := f.NewStyle(&excelize.Style{NumFmt: 49})
	_ = f.SetColStyle(sheet, "B", textStyle)
	bold, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	_ = f.SetRowStyle(sheet, 1, 1, bold)
	_ = f.SetColWidth(sheet, "A", "A", 36)
	_ = f.SetColWidth(sheet, "B", "D", 18)
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExportXLSX menulis daftar tamu ke Excel.
func ExportXLSX(header []string, rows [][]string) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Daftar Tamu"
	if err := f.SetSheetName("Sheet1", sheet); err != nil {
		return nil, err
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return nil, err
	}
	toAny := func(s []string) []any {
		out := make([]any, len(s))
		for i, v := range s {
			out[i] = v
		}
		return out
	}
	if err := sw.SetRow("A1", toAny(header)); err != nil {
		return nil, err
	}
	for i, r := range rows {
		cellRef, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := sw.SetRow(cellRef, toAny(r)); err != nil {
			return nil, err
		}
	}
	if err := sw.Flush(); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
