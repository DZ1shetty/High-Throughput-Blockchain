package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// Wallet holds an ed25519 key pair as hex-encoded strings.
type Wallet struct {
	PrivateKeyHex string
	PublicKeyHex  string
}

// NewWallet generates a fresh ed25519 key pair and returns a Wallet.
func NewWallet() (*Wallet, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("NewWallet: key generation failed: %w", err)
	}
	return &Wallet{
		PrivateKeyHex: hex.EncodeToString(priv),
		PublicKeyHex:  hex.EncodeToString(pub),
	}, nil
}

// sigPayload returns the deterministic byte slice that is signed and verified.
// It covers all immutable transaction fields, deliberately excluding
// PublicKey and Signature themselves.
func sigPayload(tx Transaction) []byte {
	return []byte(fmt.Sprintf("%s|%s|%s|%.18g|%s",
		tx.TxID, tx.Sender, tx.Receiver, tx.Amount, tx.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z")))
}

// Sign computes an ed25519 signature over the transaction payload and sets
// tx.PublicKey and tx.Signature in place.
func (w *Wallet) Sign(tx *Transaction) error {
	privBytes, err := hex.DecodeString(w.PrivateKeyHex)
	if err != nil {
		return fmt.Errorf("Sign: invalid private key hex: %w", err)
	}
	priv := ed25519.PrivateKey(privBytes)

	sig := ed25519.Sign(priv, sigPayload(*tx))
	tx.PublicKey = w.PublicKeyHex
	tx.Signature = hex.EncodeToString(sig)
	return nil
}

// Verify checks the ed25519 signature on tx.
// Returns true for valid signatures.
// Returns true for legacy transactions where Signature is empty (backward compat).
// Returns false if the public key or signature hex is malformed, or if the
// signature does not match the transaction payload.
func Verify(tx Transaction) bool {
	// Legacy transactions (pre-signature) are treated as valid.
	if tx.Signature == "" {
		return true
	}

	pubBytes, err := hex.DecodeString(tx.PublicKey)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}

	sigBytes, err := hex.DecodeString(tx.Signature)
	if err != nil {
		return false
	}

	return ed25519.Verify(ed25519.PublicKey(pubBytes), sigPayload(tx), sigBytes)
}
