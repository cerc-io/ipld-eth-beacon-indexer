package sql

import (
	"context"
	"io"
	"time"
)

// Database interfaces required by the sql indexer
type Database interface {
	Driver
}

// Driver interface has all the methods required by a driver implementation to support the sql indexer
type Driver interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) ScannableRow
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)
	Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Get(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	Begin(ctx context.Context) (Tx, error)
	Stats() Stats
	Context() context.Context
	io.Closer
}

// Tx interface to accommodate different concrete SQL transaction types
type Tx interface {
	QueryRow(ctx context.Context, sql string, args ...interface{}) ScannableRow
	Exec(ctx context.Context, sql string, args ...interface{}) (Result, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// ScannableRow interface to accommodate different concrete row types
type ScannableRow interface {
	Scan(dest ...interface{}) error
}

// Result interface to accommodate different concrete result types
type Result interface {
	RowsAffected() (int64, error)
}

// Stats interface to accommodate different concrete sql stats types
type Stats interface {
	MaxOpen() int64
	Open() int64
	InUse() int64
	Idle() int64
	WaitCount() int64
	WaitDuration() time.Duration
	MaxIdleClosed() int64
	MaxLifetimeClosed() int64
}
