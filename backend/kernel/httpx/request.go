package httpx

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"
	"strings"

	"undangan/kernel/apperror"
)

// Decode membaca body JSON (maks 1 MB).
func Decode(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	if err := dec.Decode(dst); err != nil {
		return apperror.BadRequest("Format JSON tidak valid")
	}
	return nil
}

func PageFrom(r *http.Request) Page {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pp, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	if p < 1 {
		p = 1
	}
	if pp < 1 || pp > 100 {
		pp = 20
	}
	return Page{Page: p, PerPage: pp}
}

// ParseMultipart membatasi ukuran body lalu mem-parse form (file besar di-spill ke disk sementara).
func ParseMultipart(w http.ResponseWriter, r *http.Request, maxBytes int64) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return apperror.BadRequest(fmt.Sprintf("File terlalu besar (maks %d MB) atau form tidak valid", maxBytes>>20))
	}
	return nil
}

// FormFile membaca field "file" (panggil ParseMultipart dulu).
func FormFile(r *http.Request, maxBytes int64) ([]byte, *multipart.FileHeader, error) {
	f, hdr, err := r.FormFile("file")
	if err != nil {
		return nil, nil, apperror.BadRequest("Field 'file' wajib diisi")
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		return nil, nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, nil, apperror.BadRequest(fmt.Sprintf("File terlalu besar (maks %d MB)", maxBytes>>20))
	}
	return data, hdr, nil
}

// ClientIP: X-Forwarded-For hanya dipercaya bila koneksi dari loopback (Caddy di host yang sama).
func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			first, _, _ := strings.Cut(xff, ",")
			return strings.TrimSpace(first)
		}
	}
	return host
}
