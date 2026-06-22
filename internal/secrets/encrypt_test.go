// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

// TestIsStalePublicKeyErrorRequiresConflictAndMessage verifies exact retry matching.
func TestIsStalePublicKeyErrorRequiresConflictAndMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "stale conflict", err: errInvalidEnvelopeForTest("Conflict: stale public key"), want: true},
		{name: "conflict only", err: errInvalidEnvelopeForTest("Conflict: object changed")},
		{name: "stale only", err: errInvalidEnvelopeForTest("stale public key")},
		{name: "unrelated", err: errInvalidEnvelopeForTest("forbidden")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsStalePublicKeyError(tt.err); got != tt.want {
				t.Fatalf("IsStalePublicKeyError(%q) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

// TestAssociatedData verifies the stable domain-separated write context.
func TestAssociatedData(t *testing.T) {
	t.Parallel()
	opts := WriteOptions{ClusterID: "cluster-a", Namespace: "default", KeyspaceName: "openbao-local.web", Key: "api-key"}
	got := string(AssociatedData(opts))
	want := strings.Join([]string{Algorithm, "cluster-a", "default", "openbao-local.web", "api-key"}, "\x00")
	if got != want {
		t.Fatalf("AssociatedData() = %q, want %q", got, want)
	}
}

// TestEncryptWriteValueCanBeOpenedByRecipient verifies envelope shape and ciphertext contents.
func TestEncryptWriteValueCanBeOpenedByRecipient(t *testing.T) {
	t.Parallel()
	recipientKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	recipient := base64.RawURLEncoding.EncodeToString(recipientKey.PublicKey().Bytes())
	associatedData := []byte("associated-data")
	plaintext := []byte("correct horse battery staple")

	sealed, err := EncryptWriteValue(recipient, plaintext, associatedData)
	if err != nil {
		t.Fatalf("EncryptWriteValue() error = %v", err)
	}
	opened := decryptWriteValueForTest(t, recipientKey, sealed, associatedData)
	if string(opened) != string(plaintext) {
		t.Fatalf("decrypted plaintext = %q, want %q", opened, plaintext)
	}
	if _, err := openWriteEnvelopeForTest(recipientKey, sealed, []byte("wrong-associated-data")); err == nil {
		t.Fatalf("decrypt with wrong associated data succeeded")
	}
}

// TestEncryptWriteValueRejectsInvalidPublicKey verifies recipient public key validation.
func TestEncryptWriteValueRejectsInvalidPublicKey(t *testing.T) {
	t.Parallel()
	if _, err := EncryptWriteValue("not-base64", []byte("value"), nil); err == nil {
		t.Fatalf("EncryptWriteValue() error = nil, want error")
	}
}

// decryptWriteValueForTest opens an encrypted write value and fails the test on errors.
func decryptWriteValueForTest(t *testing.T, recipientKey *ecdh.PrivateKey, sealed string, associatedData []byte) []byte {
	t.Helper()
	plaintext, err := openWriteEnvelopeForTest(recipientKey, sealed, associatedData)
	if err != nil {
		t.Fatalf("openWriteEnvelopeForTest() error = %v", err)
	}
	return plaintext
}

// openWriteEnvelopeForTest decrypts an encrypted write envelope with the recipient private key.
func openWriteEnvelopeForTest(recipientKey *ecdh.PrivateKey, sealed string, associatedData []byte) ([]byte, error) {
	data, ok := strings.CutPrefix(sealed, "podplane-secrets-v1.")
	if !ok {
		return nil, errInvalidEnvelopeForTest("missing prefix")
	}
	envelopeData, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	var envelope encryptedSecretEnvelope
	if err := json.Unmarshal(envelopeData, &envelope); err != nil {
		return nil, err
	}
	if envelope.Version != "podplane-secrets-v1" {
		return nil, errInvalidEnvelopeForTest("wrong version")
	}
	ephemeralBytes, err := base64.RawURLEncoding.DecodeString(envelope.EphemeralPublicKey)
	if err != nil {
		return nil, err
	}
	ephemeralKey, err := ecdh.X25519().NewPublicKey(ephemeralBytes)
	if err != nil {
		return nil, err
	}
	sharedSecret, err := recipientKey.ECDH(ephemeralKey)
	if err != nil {
		return nil, err
	}
	aeadKey, err := hkdf.Key(sha256.New, sharedSecret, ephemeralBytes, "podplane secrets v1", 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(aeadKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.RawURLEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, err
	}
	return gcm.Open(nil, nonce, ciphertext, associatedData)
}

// errInvalidEnvelopeForTest is a lightweight sentinel error for malformed test envelopes.
type errInvalidEnvelopeForTest string

// Error returns the invalid envelope test error string.
func (e errInvalidEnvelopeForTest) Error() string { return string(e) }
