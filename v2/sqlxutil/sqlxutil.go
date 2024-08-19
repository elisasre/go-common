package sqlxutil

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const (
	ErrNotFound StaticError = "not found"
	ErrConflict StaticError = "conflict"
)

type StaticError string

func (e StaticError) Error() string { return string(e) }

func Now() time.Time {
	return time.Now().Round(time.Millisecond).UTC()
}

// Model can be embedded in other structs to provide common fields.
type Model struct {
	ID        uint64     `json:"id" db:"id"`
	CreatedAt time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt *time.Time `json:"-" db:"deleted_at"`
}

// Create sets CreatedAt and UpdatedAt fields to current time.
func (m *Model) Create() {
	m.CreatedAt = Now()
	m.UpdatedAt = Now()
	m.DeletedAt = nil
}

// Update sets UpdatedAt field to current time.
func (m *Model) Update() {
	m.UpdatedAt = Now()
}

// IDRef returns a pointer to the ID field.
func (m *Model) IDRef() *uint64 {
	return &m.ID
}

// NotFoundWrap converts sql.ErrNoRows to ErrNotFound.
func NotFoundWrap(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// ConflictWrap converts pq's unique_violation errors to ErrConflict.
func ConflictWrap(err error) error {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) && pgErr.Code.Name() == "unique_violation" {
		return ErrConflict
	}
	return err
}

type Creatable interface {
	Create()
	IDRef() *uint64
}

type Creator interface {
	PrepareNamedContext(context.Context, string) (*sqlx.NamedStmt, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

func CreateNamed(ctx context.Context, c Creator, m Creatable, query string) error {
	m.Create()
	stmt, err := c.PrepareNamedContext(ctx, query)
	if err != nil {
		return fmt.Errorf("preparing create statement failed: %w", err)
	}
	defer func() {
		if err := stmt.Close(); err != nil {
			slog.Error("closing create statement failed", slog.String("error", err.Error()))
		}
	}()

	if err := stmt.GetContext(ctx, m.IDRef(), m); err != nil {
		return fmt.Errorf("executing create statement failed: %w", err)
	}

	return nil
}

func DeleteFromTable(ctx context.Context, e sqlx.ExecerContext, id uint64, table string) error {
	query := fmt.Sprintf(
		`UPDATE %s SET
			deleted_at = NOW()
		WHERE
			id = $1
		AND
			deleted_at IS NULL`,
		pq.QuoteIdentifier(table),
	)
	if _, err := e.ExecContext(ctx, query, id); err != nil {
		return fmt.Errorf("deleting row with id %d from %s failed: %w", id, table, err)
	}

	return nil
}

func WithTx(ctx context.Context, db *sqlx.DB, fn func(ctx context.Context, tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("starting transaction failed: %w", err)
	}

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rolling back transaction failed: %w", rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction failed: %w", err)
	}

	return nil
}

type Preparer interface {
	PrepareNamedContext(ctx context.Context, query string) (*sqlx.NamedStmt, error)
}

type DynamicQuery struct {
	conditions []string
	sortFields []string
	sortDesc   bool
	limit      int
}

func (dq *DynamicQuery) Copy() DynamicQuery {
	return DynamicQuery{
		conditions: append([]string(nil), dq.conditions...),
		sortFields: append([]string(nil), dq.sortFields...),
		sortDesc:   dq.sortDesc,
		limit:      dq.limit,
	}
}

func DynamicSelect(ctx context.Context, p Preparer, baseQuery string, dq DynamicQuery, args map[string]any, target any) error {
	q := buildQuery(baseQuery, dq)
	stmt, err := p.PrepareNamedContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if err := stmt.SelectContext(ctx, target, args); err != nil {
		return err
	}
	return err
}

func DynamicGet(ctx context.Context, p Preparer, baseQuery string, dq DynamicQuery, args map[string]any, target any) error {
	q := buildQuery(baseQuery, dq)
	stmt, err := p.PrepareNamedContext(ctx, q)
	if err != nil {
		return err
	}
	defer stmt.Close()

	if err := stmt.GetContext(ctx, target, args); err != nil {
		return err
	}
	return err
}

func buildQuery(baseQuery string, dq DynamicQuery) string {
	r := strings.NewReplacer(
		"{conditions}", strings.Join(dq.conditions, " AND "),
		"{sortFields}", strings.Join(dq.sortFields, ","),
		"{limit}", getLimit(dq.limit),
		"{sortDesc}", getSortDirection(!dq.sortDesc))
	return r.Replace(baseQuery)
}

func getLimit(limit int) string {
	if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	}
	return ""
}

func getSortDirection(sortAsc bool) string {
	if sortAsc {
		return "asc"
	}
	return "desc"
}
