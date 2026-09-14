// Package domain (audit) = jejak aksi admin/sistem.
package domain

import (
	"context"
	"time"
)

type Log struct {
	ID        int64          `json:"id"`
	ActorID   *string        `json:"actor_id"`
	ActorName *string        `json:"actor_name"`
	Action    string         `json:"action"` // "order.approve"
	Target    string         `json:"target"` // "order:<uuid>"
	Meta      map[string]any `json:"meta"`
	CreatedAt time.Time      `json:"created_at"`
}

type Repository interface {
	Insert(ctx context.Context, l *Log) error
	List(ctx context.Context, limit, offset int) ([]Log, int, error)
}

// Recorder = port yang dipakai service modul lain untuk mencatat aksi.
// Gagal mencatat tidak boleh menggagalkan aksi utama → tidak mengembalikan error.
type Recorder interface {
	Record(ctx context.Context, actorID, action, target string, meta map[string]any)
}
