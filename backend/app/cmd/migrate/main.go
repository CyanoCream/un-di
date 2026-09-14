// migrate = alat migrasi database tanpa perlu psql (lintas OS).
//
//	go run ./app/cmd/migrate up       jalankan migrasi yang belum pernah jalan
//	go run ./app/cmd/migrate status   daftar versi sudah/belum dijalankan
//	go run ./app/cmd/migrate print    cetak seluruh SQL (untuk dijalankan manual / review DBA)
//
// Membaca DATABASE_URL dari env atau file .env di direktori kerja.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sort"
	"time"

	"undangan/app/internal/config"
	"undangan/kernel/database"
	"undangan/migrations"
)

func main() {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if err := run(cmd); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(cmd string) error {
	if cmd == "print" {
		files, err := fs.Glob(migrations.FS, "*.sql")
		if err != nil {
			return err
		}
		sort.Strings(files)
		for _, f := range files {
			b, err := fs.ReadFile(migrations.FS, f)
			if err != nil {
				return err
			}
			fmt.Printf("-- ===== %s =====\nBEGIN;\n%s\nINSERT INTO schema_migrations (version) VALUES ('%s') ON CONFLICT DO NOTHING;\nCOMMIT;\n\n",
				f, b, f[:len(f)-len(".sql")])
		}
		return nil
	}

	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	switch cmd {
	case "up":
		if err := database.Migrate(ctx, pool, migrations.FS, log); err != nil {
			return err
		}
		fmt.Println("migrasi selesai")
	case "status":
		applied, pending, err := database.Status(ctx, pool, migrations.FS)
		if err != nil {
			return err
		}
		for _, v := range applied {
			fmt.Println("[x]", v)
		}
		for _, v := range pending {
			fmt.Println("[ ]", v)
		}
		fmt.Printf("%d sudah, %d belum\n", len(applied), len(pending))
	default:
		return fmt.Errorf("perintah tidak dikenal %q (pakai: up | status | print)", cmd)
	}
	return nil
}
