// Package repository (billing) = implementasi Postgres untuk kontrak repository di billing/domain.
package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/apperror"
	"undangan/kernel/database"
	"undangan/services/billing/domain"
)

// notFound: tidak ada baris atau ID bukan UUID valid (22P02) → apperror.ErrNotFound.
func notFound(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return apperror.ErrNotFound
	}
	return database.NotFound(err)
}

func execOne(ctx context.Context, pool *pgxpool.Pool, sql string, args ...any) error {
	tag, err := database.Conn(ctx, pool).Exec(ctx, sql, args...)
	if err != nil {
		return notFound(err)
	}
	if tag.RowsAffected() == 0 {
		return apperror.ErrNotFound
	}
	return nil
}

// ---- advisory lock ----

type Locker struct{ pool *pgxpool.Pool }

var _ domain.Locker = (*Locker)(nil)

func NewLocker(pool *pgxpool.Pool) *Locker { return &Locker{pool: pool} }

func (l *Locker) LockTx(ctx context.Context, key string) error {
	_, err := database.Conn(ctx, l.pool).Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, key)
	return err
}

func (l *Locker) TryLock(ctx context.Context, key string) (func(), bool, error) {
	conn, err := l.pool.Acquire(ctx)
	if err != nil {
		return nil, false, err
	}
	var ok bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(hashtextextended($1, 0))`, key).Scan(&ok); err != nil {
		conn.Release()
		return nil, false, err
	}
	if !ok {
		conn.Release()
		return nil, false, nil
	}
	release := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, err := conn.Exec(ctx, `SELECT pg_advisory_unlock(hashtextextended($1, 0))`, key); err != nil {
			// Lock sesi tidak bisa dilepas → tutup koneksi supaya lock ikut hilang.
			_ = conn.Conn().Close(ctx)
		}
		conn.Release()
	}
	return release, true, nil
}
