package auth

import (
	"testing"
	"time"
)

func TestGenerateToken_WithJTI(t *testing.T) {
	secret := "test-secret"
	userID := "test-user-id"
	role := "USER"
	ttl := 24 * time.Hour

	token, jti, err := GenerateToken(secret, userID, role, ttl)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Error("Expected token to be generated")
	}

	if jti == "" {
		t.Error("Expected JTI to be generated")
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("Expected no error parsing token, got %v", err)
	}

	if claims.ID != jti {
		t.Errorf("Expected JTI %s, got %s", jti, claims.ID)
	}

	if claims.Sub != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.Sub)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}
}

func TestParseToken_ValidToken(t *testing.T) {
	secret := "test-secret"
	userID := "test-user-id"
	role := "USER"
	ttl := 24 * time.Hour

	token, _, err := GenerateToken(secret, userID, role, ttl)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	claims, err := ParseToken(secret, token)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if claims.Sub != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.Sub)
	}

	if claims.Role != role {
		t.Errorf("Expected role %s, got %s", role, claims.Role)
	}

	if claims.ID == "" {
		t.Error("Expected JTI to be present in claims")
	}
}

func TestParseToken_InvalidToken(t *testing.T) {
	secret := "test-secret"
	invalidToken := "invalid.token.here"

	_, err := ParseToken(secret, invalidToken)
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestGenerateToken_UniqueJTIs(t *testing.T) {
	secret := "test-secret"
	userID := "test-user-id"
	role := "USER"
	ttl := 24 * time.Hour

	token1, jti1, err1 := GenerateToken(secret, userID, role, ttl)
	token2, jti2, err2 := GenerateToken(secret, userID, role, ttl)

	if err1 != nil || err2 != nil {
		t.Fatalf("Expected no errors, got %v, %v", err1, err2)
	}

	if jti1 == jti2 {
		t.Error("Expected unique JTIs for different tokens")
	}

	if token1 == token2 {
		t.Error("Expected different tokens")
	}
}
