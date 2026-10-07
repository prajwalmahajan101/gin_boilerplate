package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const MaxPageSize = 1000

type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type Repository[T Model] struct {
	db Querier
}

func NewRepository[T Model](db Querier) *Repository[T] {
	return &Repository[T]{db: db}
}

func (r *Repository[T]) WithTx(tx pgx.Tx) *Repository[T] {
	return &Repository[T]{db: tx}
}

func (r *Repository[T]) table() string {
	var zero T
	return zero.TableName()
}

func (r *Repository[T]) GetByID(ctx context.Context, id int64) (T, bool, error) {
	return r.getOne(ctx, "SELECT * FROM "+r.table()+" WHERE id = $1", id)
}

func (r *Repository[T]) GetActiveByID(ctx context.Context, id int64) (T, bool, error) {
	return r.getOne(ctx, "SELECT * FROM "+r.table()+" WHERE id = $1 AND is_active", id)
}

func (r *Repository[T]) getOne(ctx context.Context, sql string, args ...any) (T, bool, error) {
	var zero T
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return zero, false, fmt.Errorf("store: query %s: %w", r.table(), err)
	}
	rec, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByNameLax[T])
	if errors.Is(err, pgx.ErrNoRows) {
		return zero, false, nil
	}
	if err != nil {
		return zero, false, fmt.Errorf("store: scan %s: %w", r.table(), err)
	}
	return rec, true, nil
}

func (r *Repository[T]) List(ctx context.Context, limit, offset int) ([]T, error) {
	rows, err := r.db.Query(ctx, "SELECT * FROM "+r.table()+" ORDER BY id LIMIT $1 OFFSET $2", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("store: list %s: %w", r.table(), err)
	}
	recs, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		return nil, fmt.Errorf("store: scan %s: %w", r.table(), err)
	}
	return recs, nil
}

func (r *Repository[T]) Paginate(ctx context.Context, page, pageSize int) ([]T, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 1
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	offset64 := int64(page-1) * int64(pageSize)
	if offset64 > math.MaxInt32 {
		return nil, 0, fmt.Errorf("store: page %d out of range", page)
	}

	items, err := r.List(ctx, pageSize, int(offset64))
	if err != nil {
		return nil, 0, err
	}
	total, err := r.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *Repository[T]) Create(ctx context.Context, m *T) error {
	rv := reflect.ValueOf(m).Elem()
	now := time.Now().UTC()
	setCol(rv, "is_active", true)
	setCol(rv, "created_at", now)
	setCol(rv, "updated_at", now)

	var cols []string
	var placeholders []string
	var args []any
	for _, fc := range dbFields(rv) {
		if fc.col == "id" {
			continue
		}
		cols = append(cols, fc.col)
		placeholders = append(placeholders, "$"+strconv.Itoa(len(args)+1))
		args = append(args, fc.val.Interface())
	}
	sql := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
		r.table(), strings.Join(cols, ", "), strings.Join(placeholders, ", "))

	var id int64
	if err := r.db.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
		return fmt.Errorf("store: create %s: %w", r.table(), err)
	}
	setCol(rv, "id", id)
	return nil
}

func (r *Repository[T]) Update(ctx context.Context, m *T) error {
	rv := reflect.ValueOf(m).Elem()
	setCol(rv, "updated_at", time.Now().UTC())

	var sets []string
	var args []any
	var id any
	for _, fc := range dbFields(rv) {
		if fc.col == "id" {
			id = fc.val.Interface()
			continue
		}
		if fc.col == "created_at" {
			continue
		}
		sets = append(sets, fc.col+" = $"+strconv.Itoa(len(args)+1))
		args = append(args, fc.val.Interface())
	}
	args = append(args, id)
	sql := fmt.Sprintf("UPDATE %s SET %s WHERE id = $%d", r.table(), strings.Join(sets, ", "), len(args))
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("store: update %s: %w", r.table(), err)
	}
	return nil
}

func (r *Repository[T]) SoftDelete(ctx context.Context, id int64) error {
	sql := "UPDATE " + r.table() + " SET is_active = false, updated_at = now() WHERE id = $1"
	if _, err := r.db.Exec(ctx, sql, id); err != nil {
		return fmt.Errorf("store: soft-delete %s: %w", r.table(), err)
	}
	return nil
}

func (r *Repository[T]) HardDelete(ctx context.Context, id int64) error {
	sql := "DELETE FROM " + r.table() + " WHERE id = $1"
	if _, err := r.db.Exec(ctx, sql, id); err != nil {
		return fmt.Errorf("store: hard-delete %s: %w", r.table(), err)
	}
	return nil
}

func (r *Repository[T]) Exists(ctx context.Context, id int64) (bool, error) {
	var exists bool
	sql := "SELECT EXISTS(SELECT 1 FROM " + r.table() + " WHERE id = $1)"
	if err := r.db.QueryRow(ctx, sql, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("store: exists %s: %w", r.table(), err)
	}
	return exists, nil
}

func (r *Repository[T]) Count(ctx context.Context) (int, error) {
	var total int
	if err := r.db.QueryRow(ctx, "SELECT count(*) FROM "+r.table()).Scan(&total); err != nil {
		return 0, fmt.Errorf("store: count %s: %w", r.table(), err)
	}
	return total, nil
}

// SafeOrderColumn returns col if whitelisted, else def. Guards ORDER BY
// against injection when a query interpolates a caller-supplied sort column.
func SafeOrderColumn(col string, allowed map[string]struct{}, def string) string {
	if _, ok := allowed[col]; ok {
		return col
	}
	return def
}

type fieldCol struct {
	col string
	val reflect.Value
}

func dbFields(v reflect.Value) []fieldCol {
	var out []fieldCol
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if f.Anonymous && f.Type.Kind() == reflect.Struct {
			out = append(out, dbFields(v.Field(i))...)
			continue
		}
		tag := f.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		out = append(out, fieldCol{col: tag, val: v.Field(i)})
	}
	return out
}

func setCol(v reflect.Value, col string, val any) {
	for _, fc := range dbFields(v) {
		if fc.col == col && fc.val.CanSet() {
			fc.val.Set(reflect.ValueOf(val))
			return
		}
	}
}
