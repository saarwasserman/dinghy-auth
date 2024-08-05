package data

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type PasswordModel struct {
	DB *sql.DB
}

func (m PasswordModel) GetPasswordForUserId(userID int64) (string, error) {
	query := `
		SELECT password_hash
		FROM passwords
		WHERE user_id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var password_hash string

	err := m.DB.QueryRowContext(ctx, query, userID).Scan(&password_hash)
	if err != nil {
		return "", err
	}

	return password_hash, nil
}

func (m PasswordModel) CreatePasswordForUserId(userID int64, password_hash []byte) error {
	query := `
		INSERT INTO credentials (user_id, password_hash)
		VALUES ($1, $2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, userID, password_hash)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}


func (m PasswordModel) UpdatePasswordForUserId(userID int64, password_hash string) error {
	query := `
		UPDATE credentials
		SET password_hash = $1
		WHERE user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, password_hash, userID).Err()
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrRecordNotFound
		default:
			return err
		}
	}

	return nil
}

