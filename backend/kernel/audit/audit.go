// Package audit = port pencatat jejak aksi. Diimplementasikan service audit; dipakai semua service
// tanpa bergantung ke modulnya (siap dipecah jadi microservice: cukup ganti implementasi dengan client HTTP/queue).
package audit

import "context"

// Recorder mencatat aksi. Gagal mencatat tidak boleh menggagalkan aksi utama → tanpa error.
type Recorder interface {
	Record(ctx context.Context, actorID, action, target string, meta map[string]any)
}
