// Package auth handles RSA key generation, AES-128-CFB8 stream ciphers,
// and Mojang session server communication for player authentication.
package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
)

const rsaKeyBits = 1024

// GenerateKeyPair generates a fresh RSA-1024 private key.
// This matches the key size used by vanilla Minecraft.
// The key is generated once at server startup and reused for all connections.
func GenerateKeyPair() (*rsa.PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return nil, fmt.Errorf("auth: generating RSA key pair: %w", err)
	}
	return key, nil
}

// MarshalPublicKeyDER returns the DER-encoded SubjectPublicKeyInfo (X.509)
// representation of pub. This is the format sent in the Encryption Request packet.
func MarshalPublicKeyDER(pub *rsa.PublicKey) ([]byte, error) {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("auth: marshalling public key: %w", err)
	}
	return der, nil
}

// DecryptPKCS1v15 decrypts data that was encrypted by the Minecraft client
// using the server's public key with PKCS#1 v1.5 padding.
// Used to recover the shared secret and verify token from the Encryption Response.
func DecryptPKCS1v15(privKey *rsa.PrivateKey, data []byte) ([]byte, error) {
	plain, err := rsa.DecryptPKCS1v15(rand.Reader, privKey, data)
	if err != nil {
		return nil, fmt.Errorf("auth: decrypting RSA data: %w", err)
	}
	return plain, nil
}
