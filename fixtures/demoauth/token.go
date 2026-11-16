package demoauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
)

// demoSigningKey is a fixture string, not a secret.
const demoSigningKey = "demo-signing-key-not-secret"

// Claims are the demo token claims.
type Claims struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
}

// buildClaims copies the user id and email into claims.
func buildClaims(user User) Claims {
	return Claims{Subject: user.ID, Email: user.Email}
}

// signJWT returns a short fake token. It does not issue a real JWT.
func signJWT(claims Claims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(demoSigningKey))
	_, _ = mac.Write(payload)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if len(sig) > 8 {
		sig = sig[:8]
	}
	return "demo." + sig, nil
}
