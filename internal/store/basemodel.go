package store

import "time"

type BaseModel struct {
	ID        int64     `db:"id"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Model interface {
	TableName() string
}
