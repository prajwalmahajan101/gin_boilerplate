package auth

import (
	"time"

	"github.com/prajwalmahajan101/gin_boilerplate/internal/store"
)

type User struct {
	store.BaseModel
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	Role         string     `db:"role"`
	LastLoginAt  *time.Time `db:"last_login_at"`
}

func (User) TableName() string { return "users" }

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)
