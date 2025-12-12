package util_test

import (
	"testing"
	"time"

	"github.com/free-ran-ue/free-ran-ue/util"
)

var testJWTCases = []struct {
	name        string
	secret      string
	subject     string
	expiresIn   time.Duration
	extraClaims map[string]any
}{
	{
		name:      "createAndValidateJWT",
		secret:    "testSecret",
		subject:   "testSubject",
		expiresIn: 1 * time.Hour,
		extraClaims: map[string]any{
			"testKey": "testValue",
		},
	},
}

func TestCreateAndValidateJWT(t *testing.T) {
	for _, tc := range testJWTCases {
		t.Run(tc.name, func(t *testing.T) {
			token, err := util.CreateJWT(tc.secret, tc.subject, tc.expiresIn, tc.extraClaims)
			if err != nil {
				t.Fatalf("Failed to create JWT: %v", err)
			}

			claims, err := util.ValidateJWT(token, tc.secret)
			if err != nil {
				t.Fatalf("Failed to validate JWT: %v", err)
			}

			if claims["sub"] != tc.subject {
				t.Fatalf("Expected subject %s, got %s", tc.subject, claims["sub"])
			}

			for k, v := range tc.extraClaims {
				if claims[k] != v {
					t.Fatalf("Expected claim %s to be %v, got %v", k, v, claims[k])
				}
			}
		})
	}
}
