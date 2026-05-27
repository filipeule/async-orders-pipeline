package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// DecryptAES256GCM decrypts an AEAD payload produced by AES-256-GCM.
//
// Expected inputs:
//   - hexKey:        32-byte key encoded as 64 hex chars.
//   - nonceB64:      12-byte nonce encoded in base64. MUST be unique per
//                    message for a given key — nonce reuse is catastrophic
//                    in GCM (leaks the authentication subkey).
//   - ciphertextB64: ciphertext concatenated with the 16-byte auth tag,
//                    encoded in base64. This is how Go's cipher.AEAD.Seal
//                    formats its output, and what AEAD.Open expects.
//
// On any failure (wrong key, tampered ciphertext, malformed nonce, etc.)
// returns an error and zero plaintext — never a partial decrypt.
func DecryptAES256GCM(hexKey, nonceB64, ciphertextB64 string) ([]byte, error) {
	key, err := hex.DecodeString(hexKey)
	if err != nil {
		return nil, errors.New("invalid key hex: " + err.Error())
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("invalid key length: got %d bytes, want 32", len(key))
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceB64)
	if err != nil {
		return nil, errors.New("invalid nonce base64: " + err.Error())
	}

	ct, err := base64.StdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return nil, errors.New("invalid ciphertext base64: " + err.Error())
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(nonce) != aesgcm.NonceSize() {
		return nil, fmt.Errorf("invalid nonce length: got %d, want %d", len(nonce), aesgcm.NonceSize())
	}

	// Open runs decryption and tag verification atomically. If the tag does
	// not match (tampering, wrong key, wrong nonce), it returns an error
	// without leaking any plaintext bytes.
	plaintext, err := aesgcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm authentication failed: %w", err)
	}

	return plaintext, nil
}
