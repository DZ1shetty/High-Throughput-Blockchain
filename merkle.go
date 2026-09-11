package main

import (
	"crypto/sha256"
	"fmt"
)

// ComputeMerkleRoot builds a Merkle tree from the TxID of each transaction
// and returns the root hash as a lowercase hex string.
// Returns an empty string if there are no transactions.
func ComputeMerkleRoot(transactions []Transaction) string {
	if len(transactions) == 0 {
		return ""
	}

	// Build the leaf layer: SHA-256 of each TxID.
	layer := make([]string, len(transactions))
	for i, tx := range transactions {
		sum := sha256.Sum256([]byte(tx.TxID))
		layer[i] = fmt.Sprintf("%x", sum)
	}

	// Repeatedly hash adjacent pairs until one node remains.
	for len(layer) > 1 {
		// Duplicate the last node if the layer has an odd count.
		if len(layer)%2 != 0 {
			layer = append(layer, layer[len(layer)-1])
		}

		next := make([]string, len(layer)/2)
		for i := 0; i < len(layer); i += 2 {
			combined := layer[i] + layer[i+1]
			sum := sha256.Sum256([]byte(combined))
			next[i/2] = fmt.Sprintf("%x", sum)
		}
		layer = next
	}

	return layer[0]
}
