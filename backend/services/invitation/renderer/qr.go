package renderer

import (
	"fmt"
	"html/template"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// QRSVG membuat QR code sebagai SVG inline (tanpa JS/library di sisi tamu).
// Isi QR dibuat server, sehingga aman ditandai template.HTML.
func QRSVG(content string) template.HTML {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return ""
	}
	q.DisableBorder = false
	bits := q.Bitmap()
	n := len(bits)
	var b strings.Builder
	fmt.Fprintf(&b, `<svg class="qr" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges" role="img" aria-label="QR code tiket"><rect width="%d" height="%d" fill="#fff"/><path fill="#000" d="`, n, n, n, n)
	for y, row := range bits {
		for x := 0; x < len(row); x++ {
			if !row[x] {
				continue
			}
			start := x
			for x < len(row) && row[x] {
				x++
			}
			fmt.Fprintf(&b, "M%d %dh%dv1h-%dz", start, y, x-start, x-start)
		}
	}
	b.WriteString(`"/></svg>`)
	return template.HTML(b.String())
}
