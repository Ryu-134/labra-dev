package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestHMACValidatorAndRBAC(t *testing.T) {
	secret := []byte("test-secret")
	validator := HMACValidator{
		Issuer:   "https://issuer.example.com",
		Audience: "labra-api",
		Secret:   secret,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":            "https://issuer.example.com",
		"aud":            "labra-api",
		"exp":            time.Now().Add(time.Hour).Unix(),
		"sub":            "user-abc",
		"user_id":        42,
		"cognito:groups": []string{"owner"},
	})
	rawToken, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	h := RequireAuth(validator)(RequireAnyRole("owner")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := PrincipalFromContext(r.Context())
		if !ok {
			t.Fatalf("missing principal")
		}
		if principal.UserID != 42 {
			t.Fatalf("expected user id 42, got %d", principal.UserID)
		}
		w.WriteHeader(http.StatusNoContent)
	})))

	req := httptest.NewRequest(http.MethodGet, "/v1/protected", nil)
	req.Header.Set("Authorization", "Bearer "+rawToken)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
}
