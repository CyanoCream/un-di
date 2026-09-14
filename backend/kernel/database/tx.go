package database

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"undangan/kernel/apperror"
)

// DBTX = subset method yang dimiliki *pgxpool.Pool maupun pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, table pgx.Identifier, columns []string, src pgx.CopyFromSource) (int64, error)
}

// TxManager menjalankan beberapa operasi repository (bisa lintas modul) dalam satu transaksi.
type TxManager interface {
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type txKey struct{}

type pgxTxManager struct{ pool *pgxpool.Pool }

func NewTxManager(pool *pgxpool.Pool) TxManager { return &pgxTxManager{pool: pool} }

func (m *pgxTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx) // sudah di dalam transaksi → gabung
	}
	return pgx.BeginFunc(ctx, m.pool, func(tx pgx.Tx) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

// Conn mengembalikan transaksi aktif di context, atau pool.
// Semua repository wajib memakai ini supaya ikut transaksi dari service.
func Conn(ctx context.Context, pool *pgxpool.Pool) DBTX {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return pool
}

// NotFound memetakan pgx.ErrNoRows dan ID berformat salah (mis. bukan UUID) → apperror.ErrNotFound.
func NotFound(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || (errors.As(err, &pgErr) && pgErr.Code == "22P02") {
		return apperror.ErrNotFound
	}
	return err
}

// IsForeignKeyViolation: referensi ke baris yang tidak ada.
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23503"
}
