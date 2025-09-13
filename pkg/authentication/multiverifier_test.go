package authentication

import (
    "encoding/base64"
    "testing"

    "golang.org/x/crypto/argon2"
)

func TestMultiVerifier_RoutesByPrefix(t *testing.T) {
    mv := NewMultiVerifier(nil, nil)

    // unixcrypt path (legacy)
    unixHash := "tek4edTZE898g" // hash for password "testpassword123" with salt "te"
    if err := mv.VerifyPassword("testpassword123", unixHash); err != nil {
        t.Fatalf("unixcrypt path should verify: %v", err)
    }
    if err := mv.VerifyPassword("wrong", unixHash); err == nil {
        t.Fatalf("unixcrypt path should fail for wrong password")
    }

    // argon2id path
    salt := []byte("0123456789abcdef")
    hash := argon2.IDKey([]byte("p@ssw0rd"), salt, 2, 64*1024, 1, 32)
    phc := "$argon2id$v=19$m=65536,t=2,p=1$" + base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(hash)
    if err := mv.VerifyPassword("p@ssw0rd", phc); err != nil {
        t.Fatalf("argon2id path should verify: %v", err)
    }
    if err := mv.VerifyPassword("nope", phc); err == nil {
        t.Fatalf("argon2id path should fail for wrong password")
    }
}

