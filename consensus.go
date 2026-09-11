package main

import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"sync"
)

// Validator is a network participant that can vote on blocks.
// It reuses the ed25519 Wallet from wallet.go.
type Validator struct {
	wallet *Wallet
}

// ValidatorSet is a fixed group of validators used for consensus.
type ValidatorSet struct {
	validators []*Validator
}

// NewValidatorSet creates n independent validators, each with their own
// ed25519 key pair. Panics if key generation fails.
func NewValidatorSet(n int) *ValidatorSet {
	if n < 1 {
		n = 1
	}
	vs := &ValidatorSet{validators: make([]*Validator, n)}
	for i := range vs.validators {
		w, err := NewWallet()
		if err != nil {
			panic(fmt.Sprintf("NewValidatorSet: failed to create validator %d: %v", i, err))
		}
		vs.validators[i] = &Validator{wallet: w}
	}
	return vs
}

// blockPayload returns the deterministic bytes that validators sign.
// Covers the block's Hash (which already commits to all content).
func blockPayload(b *Block) []byte {
	return []byte(b.Hash)
}

// ReachConsensus runs all validators in parallel. Each validator:
//  1. Verifies the block hash is non-empty.
//  2. Recomputes the Merkle root and confirms it matches the stored value.
//  3. Signs the block hash with its ed25519 key.
//
// Returns true and sets block.ConsensusSignatures if ≥ ⌈2/3⌉ validators
// sign. Returns false (Byzantine fault tolerance) otherwise.
func (vs *ValidatorSet) ReachConsensus(block *Block) bool {
	n := len(vs.validators)
	quorum := (2*n + 2) / 3 // ceiling of 2n/3

	type vote struct {
		sig string
		ok  bool
	}
	votes := make([]vote, n)
	var wg sync.WaitGroup

	for i, v := range vs.validators {
		wg.Add(1)
		go func(idx int, val *Validator) {
			defer wg.Done()

			// Check 1: block hash must be present.
			if block.Hash == "" {
				votes[idx] = vote{ok: false}
				return
			}

			// Check 2: recompute Merkle root and compare.
			if ComputeMerkleRoot(block.Transactions) != block.MerkleRoot {
				votes[idx] = vote{ok: false}
				return
			}

			// Sign the block hash with this validator's private key.
			privBytes, err := hex.DecodeString(val.wallet.PrivateKeyHex)
			if err != nil || len(privBytes) != ed25519.PrivateKeySize {
				votes[idx] = vote{ok: false}
				return
			}
			sig := ed25519.Sign(ed25519.PrivateKey(privBytes), blockPayload(block))
			votes[idx] = vote{sig: hex.EncodeToString(sig), ok: true}
		}(i, v)
	}

	wg.Wait()

	// Collect signatures from validators that approved.
	var sigs []string
	for _, v := range votes {
		if v.ok {
			sigs = append(sigs, v.sig)
		}
	}

	if len(sigs) < quorum {
		return false
	}

	block.ConsensusSignatures = sigs
	return true
}

// defaultValidatorSet is the package-level validator set used by AddBlock
// and AddBlockParallel. Initialised once with 4 validators.
var defaultValidatorSet = NewValidatorSet(4)
