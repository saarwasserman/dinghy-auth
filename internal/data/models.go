package data

import (
	"database/sql"
	"errors"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRecordNotFound = errors.New("record not found")
	ErrEditConflict   = errors.New("edit conflict")
)

type Models struct {
	Passwords   PasswordModel
	Permissions PermissionModel
	Tokens      TokenModel
	Users       UserModel
}

func NewModels(db *sql.DB, cache *redis.Client) Models {
	return Models{
		Passwords:   PasswordModel{DB: db},
		Permissions: PermissionModel{DB: db},
		Tokens:      TokenModel{DB: db, Cache: cache},
	}
}
