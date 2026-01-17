package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"
	role := "USER"
	ttl := time.Hour

	token, _, err := GenerateToken(secret, userID, role, ttl)

	assert.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestParseToken(t *testing.T) {
	secret := "test-secret-key"
	userID := "user-123"
	role := "USER"

	t.Run("valid token", func(t *testing.T) {
		token, _, err := GenerateToken(secret, userID, role, time.Hour)
		assert.NoError(t, err)

		claims, err := ParseToken(secret, token)
		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, userID, claims.Sub)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("invalid signature", func(t *testing.T) {
		token, _, err := GenerateToken("wrong-secret", userID, role, time.Hour)
		assert.NoError(t, err)

		claims, err := ParseToken(secret, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("expired token", func(t *testing.T) {
		c := Claims{
			Sub:  userID,
			Role: role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}
		tkn := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
		token, err := tkn.SignedString([]byte(secret))
		assert.NoError(t, err)

		claims, err := ParseToken(secret, token)
		assert.Error(t, err)
		assert.Nil(t, claims)
	})

	t.Run("malformed token", func(t *testing.T) {
		claims, err := ParseToken(secret, "not.a.valid.token")
		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestHashPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword123"

	hash, err := HashPassword(password)
	assert.NoError(t, err)

	t.Run("correct password", func(t *testing.T) {
		isValid := VerifyPassword(hash, password)
		assert.True(t, isValid)
	})

	t.Run("wrong password", func(t *testing.T) {
		isValid := VerifyPassword(hash, "wrongpassword")
		assert.False(t, isValid)
	})

	t.Run("different hash each time", func(t *testing.T) {
		hash2, err := HashPassword(password)
		assert.NoError(t, err)
		assert.NotEqual(t, hash, hash2)
		assert.True(t, VerifyPassword(hash2, password))
	})
}
