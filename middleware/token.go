package middleware

import (
	"errors"
	"strconv"
	"strings"
)

// token format: "userID_timestamp"
func ParseToken(token string) (int64, error) {
	parts := strings.Split(token, "_")
	if len(parts) != 2 {
		return 0, errors.New("invalid token")
	}

	userID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}

	return userID, nil
}