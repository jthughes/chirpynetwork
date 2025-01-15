package auth

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestValidBearerToken(t *testing.T) {

	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("2s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}

	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	result, err := GetBearerToken(header)
	if err != nil {
		t.Fatalf("Failed to get bearer token: %v", err)
	}
	if result != token {
		t.Fatalf("Token mismatch: have '%s' want '%s'", result, token)
	}
}

func TestInvalidBearerToken(t *testing.T) {

	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("2s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}

	header := http.Header{}
	header.Add("Authorization", fmt.Sprintf("Bearer %s123", token))

	result, err := GetBearerToken(header)
	if err != nil {
		t.Fatalf("Failed to get bearer token: %v", err)
	}
	_, err = ValidateJWT(result, tokenSecret)
	if err == nil {
		t.Fatal("Validation succeeded when it should have failed")
	}

}

func TestInvalidBearerToken2(t *testing.T) {

	userID := uuid.New()
	tokenSecret := "ThisIsATestSecret"
	expiry, _ := time.ParseDuration("2s")
	token, err := MakeJWT(userID, tokenSecret, expiry)
	if err != nil {
		t.Fatal("Failed to create token")
	}

	header := http.Header{}
	header.Add("Authorization", token)

	_, err = GetBearerToken(header)
	if err == nil {
		t.Fatalf("Should have failed to get bearer token but didn't")
	}
	expected_error := "invalid authorization header"
	if err.Error() != expected_error {
		t.Fatalf("Error should be '%s', got '%v'", expected_error, err)
	}
}

func TestMissingBearerToken(t *testing.T) {

	header := http.Header{}
	_, err := GetBearerToken(header)
	if err == nil {
		t.Fatalf("Should have failed to get bearer token but didn't")
	}
	expected_error := "missing authorization header"
	if err.Error() != expected_error {
		t.Fatalf("Error should be '%s', got '%v'", expected_error, err)
	}
}
