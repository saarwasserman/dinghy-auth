package data

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base32"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/saarwasserman/auth/internal/validator"
)

const (
	ScopeActivation     = "activation"
	ScopeAuthentication = "authentication"
)

type Token struct {
	Plaintext string    `json:"token"`
	Hash      []byte    `json:"-"`
	UserID    int64     `json:"-" redis:"userId"`
	Expiry    time.Time `json:"expiry" redis:"expiry"`
	Scope     string    `json:"-"`
}

const tokenCacheKeyPrefix = "tokens:hash"
const userTokensCacheKeyPrefix = "tokens:user"


func generateToken(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token := &Token{
		UserID: userID,
		Expiry: time.Now().Add(ttl),
		Scope:  scope,
	}

	randomBytes := make([]byte, 16)

	_, err := rand.Read(randomBytes)
	if err != nil {
		return nil, err
	}

	token.Plaintext = base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)
	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token, nil
}

func ValidateTokenPlaintext(v *validator.Validator, tokenPlaintext string) {
	v.Check(tokenPlaintext != "", "token", "must be provided")
	v.Check(len(tokenPlaintext) == 26, "token", "must be 26 bytes long")
}

type TokenModel struct {
	DB *sql.DB
	Cache *redis.Client
}

func (m TokenModel) New(userID int64, ttl time.Duration, scope string) (*Token, error) {
	token, err := generateToken(userID, ttl, scope)
	if err != nil {
		return nil, err
	}

	err = m.Insert(token)
	return token, err
}

func (m TokenModel) Insert(token *Token) error {
	query := `
		INSERT INTO tokens (hash, user_id, expiry, scope)
		VALUES ($1, $2, $3, $4)`

	args := []any{
		token.Hash,
		token.UserID,
		token.Expiry,
		token.Scope,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	// cache
	m.Cache.SAdd(context.Background(), "")

	return nil
}

func (m TokenModel) DeleteAllForUser(scope string, userID int64) error {
	query := `
		DELETE FROM tokens
		WHERE scope = $1 AND user_id = $2`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.ExecContext(ctx, query, scope, userID)
	return err
}

func (t TokenModel) GetForToken(tokenScope, tokenPlaintext string) (*Token, error) {

	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	query := `
		SELECT hash, user_id, expiry, scope 
		FROM tokens
		WHERE hash = $1
		AND tokens.scope = $2
		AND tokens.expiry > $3`

	args := []any{tokenHash[:], tokenScope, time.Now()}

	//var user User
	var token Token

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := t.DB.QueryRowContext(ctx, query, args...).Scan(
		&token.Hash,
		&token.UserID,
		&token.Expiry,
		&token.Scope)

	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &token, nil
}


// cache //
func (t TokenModel) GetTokenFromCache(tokenScope, tokenPlaintext string) (*Token) {
	
	tokenHash := sha256.Sum256([]byte(tokenPlaintext))

	token := &Token{
		Plaintext: tokenPlaintext,
		Hash: tokenHash[:],
		Scope: tokenScope,

	}

	ctx := context.Background()

	err := t.Cache.HMGet(ctx, fmt.Sprintf("%s:%s", tokenCacheKeyPrefix, tokenHash), "userId", "expiry").Scan(token)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}

	// Handle expiry, get from db
	if token.Expiry.Before(time.Now().UTC()) {
		t.Cache.Del(ctx, string(token.Hash))
		return nil
	}

	return token
}


func (t TokenModel) UpdateCache(token *Token) error {
	ctx := context.Background()

	var updateToken *redis.IntCmd
	t.Cache.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
		updateToken = pipe.HSet(context.Background(),
						 fmt.Sprintf("%s:%s", tokenCacheKeyPrefix, token.Hash),
						 "userId", token.UserID, "expiry", token.Expiry)

		pipe.LPush(ctx, fmt.Sprintf("%s:%d", userTokensCacheKeyPrefix, token.UserID), token.Hash)

		return nil
	})

	_, err := updateToken.Result()
	if err != nil {
		return err
	}

	return nil
}


func (t TokenModel) DeleteTokensCacheForUser(scope string, userId int64) (error) {
	ctx := context.Background()
	// get the list of tokens of user
	tokenHashes, err := t.Cache.LRange(ctx, fmt.Sprintf("%s:%d", userTokensCacheKeyPrefix, userId), 0, -1).Result()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	// delete token by reference
	for _, tokenHash := range tokenHashes {
		_, err := t.Cache.Del(ctx, fmt.Sprintf("%s:%s", tokenCacheKeyPrefix, tokenHash)).Result()
		if err != nil {
			fmt.Println(err.Error())
			return err
		}
	}

	_, err = t.Cache.Del(ctx, fmt.Sprintf("%s:%d", userTokensCacheKeyPrefix, userId)).Result()
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	return nil
}
