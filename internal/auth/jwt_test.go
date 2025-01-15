package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("2s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}
	result, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	if result != userID {
		t.Fatalf("Failed: Have '%s' want '%s'", result.String(), userID.String())
	}
}

func TestInvalidJWTsecret(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("2s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}
	_, err = ValidateJWT(token, "tokenSecret")
	if err == nil {
		t.Fatalf("Token valid but should be invalid")
	}
	expected_error := "signature is invalid"
	if strings.Contains(err.Error(), expected_error) != true {
		t.Fatalf("Should contain '%s', got '%v'", expected_error, err)
	}
}

func TestExpiredJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("1s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}
	result, err := ValidateJWT(token, tokenSecret)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}
	if result != userID {
		t.Fatalf("Failed: Have '%s' want '%s'", result.String(), userID.String())
	}
	time.Sleep(expiry)
	_, err = ValidateJWT(token, tokenSecret)
	if err == nil {
		t.Fatalf("Token valid but should be invalid")
	}
	expected_error := "token is expired"
	if strings.Contains(err.Error(), expected_error) != true {
		t.Fatalf("Should contain '%s', got '%v'", expected_error, err)
	}
}
