package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func MakeAccessToken(userID uuid.UUID, tokenSecret string) (string, error) {
	accessExpiry := time.Hour
	return MakeJWT(userID, tokenSecret, accessExpiry)
}

func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	now := time.Now().UTC()
	token := jwt.NewWithClaims(
		jwt.SigningMethodHS256,
		jwt.RegisteredClaims{
			Issuer:    "chirpy",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			Subject:   userID.String(),
		},
	)
	jwt, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}
	return jwt, nil
}

func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	// Parse the jwt
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(tokenSecret), nil
		},
	)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse token: %v", err)
	}

	// Checks issuer
	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to retrieve issuer from claim: %v", err)
	} else if issuer != "chirpy" {
		return uuid.UUID{}, fmt.Errorf("invalid issuer")
	}

	// Checks expiry
	expiry, err := token.Claims.GetExpirationTime()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to retrieve expiration time from claim: %v", err)
	} else if expiry.Before(time.Now().UTC()) {
		return uuid.UUID{}, fmt.Errorf("token expired")
	}

	// Gets user id
	user_id, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to retrieve subject from claim: %v", err)
	}

	id, err := uuid.Parse(user_id)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed to parse subject into uuid: %v", err)
	}
	return id, nil
}

func MakeRefreshToken() (string, error) {
	data := [32]byte{}
	_, err := rand.Read(data[:])
	if err != nil {
		return "", err
	}
	token := hex.EncodeToString(data[:])
	return token, nil
}
