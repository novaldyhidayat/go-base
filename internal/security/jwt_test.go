package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"go-base/internal/config"
)

func TestJWTManagerEnforcesIssuerAudienceAndAlgorithm(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	privatePath := filepath.Join(dir, "private.pem")
	publicPath := filepath.Join(dir, "public.pem")
	writePEM(t, privatePath, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(privateKey))
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	writePEM(t, publicPath, "PUBLIC KEY", publicDER)

	cfg := config.JWTConfig{
		PrivateKeyPath: privatePath,
		PublicKeyPath:  publicPath,
		Issuer:         "go-base",
		Audience:       "clients",
		ExpiresIn:      time.Minute,
	}
	manager, err := NewJWTManager(cfg)
	if err != nil {
		t.Fatal(err)
	}

	token, err := manager.Sign("subject", []string{"user"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Verify(token); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}

	wrongIssuer := jwt.NewWithClaims(jwt.SigningMethodRS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "wrong",
			Audience:  jwt.ClaimStrings{"clients"},
			Subject:   "subject",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	})
	wrongIssuerToken, err := wrongIssuer.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Verify(wrongIssuerToken); err == nil {
		t.Fatal("token with wrong issuer was accepted")
	}

	hmac := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{})
	hmacToken, err := hmac.SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Verify(hmacToken); err == nil {
		t.Fatal("HMAC token was accepted")
	}
}

func writePEM(t *testing.T, path, kind string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: data}), 0o600); err != nil {
		t.Fatal(err)
	}
}
