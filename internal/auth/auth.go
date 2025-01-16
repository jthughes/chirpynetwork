package auth

import (
	"fmt"
	"net/http"
	"strings"
)

func GetBearerToken(headers http.Header) (string, error) {
	auth_header := headers.Get("Authorization")
	if auth_header == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	slice := strings.Split(auth_header, " ")
	fmt.Println(len(slice), slice[0])
	if len(slice) != 2 && slice[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header")
	}
	return slice[1], nil
}

func GetAPIKey(headers http.Header) (string, error) {
	auth_header := headers.Get("Authorization")
	if auth_header == "" {
		return "", fmt.Errorf("missing authorization header")
	}
	slice := strings.Split(auth_header, " ")
	fmt.Println(len(slice), slice[0])
	if len(slice) != 2 && slice[0] != "ApiKey" {
		return "", fmt.Errorf("invalid apikey header")
	}
	return slice[1], nil
}
