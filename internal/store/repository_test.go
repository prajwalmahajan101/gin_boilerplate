package store

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// testModel is a minimal Model for testing the generic Repository.
type testModel struct {
	BaseModel
	Name string `db:"name"`
}

func (testModel) TableName() string { return "tests" }

// --- mock Querier ---

type recorded struct {
	sql  string
	args []any
}

type mockQuerier struct {
	recorded []recorded
	rows     *mockRows
	row      *mockRow
	execTag  pgconn.CommandTag
}

func (m *mockQuerier) Query(_ context.Context, sql string, args ...any) (pgx.Rows, error) {
	m.recorded = append(m.recorded, recorded{sql, args})
	return m.rows, nil
}

func (m *mockQuerier) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	m.recorded = append(m.recorded, recorded{sql, args})
	return m.row
}

func (m *mockQuerier) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	m.recorded = append(m.recorded, recorded{sql, args})
	return m.execTag, nil
}

// mockRow implements pgx.Row — returns a canned id for RETURNING id.
type mockRow struct{ id int64 }

func (r *mockRow) Scan(dest ...any) error {
	if len(dest) > 0 {
		if p, ok := dest[0].(*int64); ok {
			*p = r.id
		}
		if p, ok := dest[0].(*bool); ok {
			*p = true
		}
		if p, ok := dest[0].(*int); ok {
			*p = int(r.id)
		}
	}
	return nil
}

// mockRows stubs pgx.Rows — returns ErrNoRows on collect (empty result set).
type mockRows struct{}

func (r *mockRows) Close()                                       {}
func (r *mockRows) Err() error                                   { return nil }
func (r *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *mockRows) Next() bool                                   { return false }
func (r *mockRows) Scan(_ ...any) error                          { return nil }
func (r *mockRows) Values() ([]any, error)                       { return nil, nil }
func (r *mockRows) RawValues() [][]byte                          { return nil }
func (r *mockRows) Conn() *pgx.Conn                              { return nil }
func (r *mockRows) TypeMap() *pgtype.Map                         { return pgtype.NewMap() }

// --- tests ---

func TestCreate_StampsFields(t *testing.T) {
	mq := &mockQuerier{row: &mockRow{id: 42}}
	repo := NewRepository[testModel](mq)

	m := testModel{Name: "foo"}
	before := time.Now().UTC()
	if err := repo.Create(context.Background(), &m); err != nil {
		t.Fatal(err)
	}

	if m.ID != 42 {
		t.Errorf("id: got %d, want 42", m.ID)
	}
	if !m.IsActive {
		t.Error("is_active should be true")
	}
	if m.CreatedAt.Before(before) || m.UpdatedAt.Before(before) {
		t.Error("timestamps not stamped")
	}
	if len(mq.recorded) != 1 {
		t.Fatalf("expected 1 query, got %d", len(mq.recorded))
	}
	sql := mq.recorded[0].sql
	if !containsStr(sql, "INSERT INTO tests") || !containsStr(sql, "RETURNING id") {
		t.Errorf("unexpected SQL: %s", sql)
	}
}

func TestGetByID_NotFound(t *testing.T) {
	mq := &mockQuerier{rows: &mockRows{}}
	repo := NewRepository[testModel](mq)

	_, found, err := repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Error("should not be found")
	}
}

func TestSoftDelete_SQL(t *testing.T) {
	mq := &mockQuerier{}
	repo := NewRepository[testModel](mq)

	if err := repo.SoftDelete(context.Background(), 7); err != nil {
		t.Fatal(err)
	}
	if len(mq.recorded) != 1 {
		t.Fatal("expected 1 exec")
	}
	sql := mq.recorded[0].sql
	if !containsStr(sql, "is_active = false") || !containsStr(sql, "id = $1") {
		t.Errorf("unexpected SQL: %s", sql)
	}
	if mq.recorded[0].args[0] != int64(7) {
		t.Errorf("arg: got %v", mq.recorded[0].args[0])
	}
}

func TestHardDelete_SQL(t *testing.T) {
	mq := &mockQuerier{}
	repo := NewRepository[testModel](mq)

	if err := repo.HardDelete(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	sql := mq.recorded[0].sql
	if !containsStr(sql, "DELETE FROM tests") {
		t.Errorf("unexpected SQL: %s", sql)
	}
}

func TestExists(t *testing.T) {
	mq := &mockQuerier{row: &mockRow{id: 1}} // Scan writes true
	repo := NewRepository[testModel](mq)

	exists, err := repo.Exists(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("should exist")
	}
}

func TestCount(t *testing.T) {
	mq := &mockQuerier{row: &mockRow{id: 5}} // Scan writes 5 into *int
	repo := NewRepository[testModel](mq)

	n, err := repo.Count(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 5 {
		t.Errorf("count: got %d, want 5", n)
	}
}

func TestPaginate_Clamps(t *testing.T) {
	mq := &mockQuerier{rows: &mockRows{}, row: &mockRow{id: 0}}
	repo := NewRepository[testModel](mq)

	// page<1 and pageSize>MaxPageSize should be clamped
	_, _, err := repo.Paginate(context.Background(), -1, 5000)
	if err != nil {
		t.Fatal(err)
	}
	// List call: offset=0, limit=MaxPageSize
	listArgs := mq.recorded[0].args
	if listArgs[0] != MaxPageSize || listArgs[1] != 0 {
		t.Errorf("clamped args: got limit=%v offset=%v", listArgs[0], listArgs[1])
	}
}

func TestDbFields_WalksEmbedded(t *testing.T) {
	m := testModel{Name: "x"}
	m.ID = 1
	m.IsActive = true
	fields := dbFields(reflect.ValueOf(m))
	var cols []string
	for _, f := range fields {
		cols = append(cols, f.col)
	}
	want := []string{"id", "is_active", "created_at", "updated_at", "name"}
	if !reflect.DeepEqual(cols, want) {
		t.Errorf("dbFields cols: got %v, want %v", cols, want)
	}
}

func TestSetCol(t *testing.T) {
	m := testModel{}
	rv := reflect.ValueOf(&m).Elem()
	setCol(rv, "name", "hello")
	if m.Name != "hello" {
		t.Errorf("setCol name: got %q", m.Name)
	}
	setCol(rv, "id", int64(99))
	if m.ID != 99 {
		t.Errorf("setCol id: got %d", m.ID)
	}
}

func TestSafeOrderColumn(t *testing.T) {
	allowed := map[string]struct{}{"id": {}, "created_at": {}}
	cases := []struct {
		col, want string
	}{
		{"created_at", "created_at"},
		{"id; DROP TABLE items", "id"},
		{"", "id"},
		{"ID", "id"},
	}
	for _, tc := range cases {
		if got := SafeOrderColumn(tc.col, allowed, "id"); got != tc.want {
			t.Errorf("SafeOrderColumn(%q) = %q, want %q", tc.col, got, tc.want)
		}
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
