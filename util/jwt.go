package util

import (
	"encoding/json"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

func CreateJWT(secret string, subject string, expiresIn time.Duration, extraClaims map[string]any) (string, error) {
	if secret == "" {
		return "", errors.New("secret must not be empty")
	}
	if expiresIn <= 0 {
		return "", errors.New("expiresIn must be positive")
	}

	now := time.Now()

	claims := jwt.MapClaims{
		"sub": subject,
		"iat": now.Unix(),
		"exp": now.Add(expiresIn).Unix(),
	}
	for k, v := range extraClaims {
		if k == "sub" || k == "iat" || k == "exp" {
			continue
		}
		claims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}
	return signedString, nil
}

func ValidateJWT(tokenString string, secret string) (jwt.MapClaims, error) {
	if secret == "" {
		return nil, errors.New("secret must not be empty")
	}
	if tokenString == "" {
		return nil, errors.New("token must not be empty")
	}

	parsedToken, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil {
		return nil, err
	}

	if !parsedToken.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := parsedToken.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims type")
	}

	if expVal, exists := claims["exp"]; exists {
		switch v := expVal.(type) {
		case float64:
			if time.Now().Unix() >= int64(v) {
				return nil, errors.New("token expired")
			}
		case json.Number:
			if unix, err := v.Int64(); err == nil {
				if time.Now().Unix() >= unix {
					return nil, errors.New("token expired")
				}
			}
		case int64:
			if time.Now().Unix() >= v {
				return nil, errors.New("token expired")
			}
		case int:
			if time.Now().Unix() >= int64(v) {
				return nil, errors.New("token expired")
			}
		}
	}

	return claims, nil
}
