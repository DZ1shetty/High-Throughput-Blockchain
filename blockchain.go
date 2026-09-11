package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// Blockchain holds an ordered slice of Blocks.
type Blockchain struct {
	Blocks []Block `json:"blocks"`
}

// NewBlockchain initialises a blockchain with a genesis block.
func NewBlockchain() *Blockchain {
	genesis := Block{
		Index:        0,
		Timestamp:    time.Now().UTC(),
		Transactions: []Transaction{},
		PreviousHash: "0",
	}
	genesis.Hash = CalculateHash(genesis)

	return &Blockchain{
		Blocks: []Block{genesis},
	}
}

// CalculateHash serialises the block to JSON and returns its SHA-256 hex digest.
// MerkleRoot is included in the hashed payload so the hash covers all tx data.
func CalculateHash(block Block) string {
	data, err := json.Marshal(struct {
		Index        int           `json:"index"`
		Timestamp    time.Time     `json:"timestamp"`
		Transactions []Transaction `json:"transactions"`
		PreviousHash string        `json:"previous_hash"`
		MerkleRoot   string        `json:"merkle_root"`
	}{
		Index:        block.Index,
		Timestamp:    block.Timestamp,
		Transactions: block.Transactions,
		PreviousHash: block.PreviousHash,
		MerkleRoot:   block.MerkleRoot,
	})
	if err != nil {
		// Fallback to a fmt-based representation so hashing never panics.
		data = []byte(fmt.Sprintf("%d%s%s%s%v",
			block.Index, block.Timestamp, block.PreviousHash, block.MerkleRoot, block.Transactions))
	}

	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

// AddBlock creates a new block containing the given transactions, links it to
// the current chain tip, and appends it to bc.Blocks.
// Transactions with an invalid signature are silently skipped.
// Transactions with an empty Signature are treated as valid legacy transactions.
// The block is only appended if it reaches validator consensus (≥2/3 sign).
func (bc *Blockchain) AddBlock(transactions []Transaction) {
	prev := bc.Blocks[len(bc.Blocks)-1]

	valid := make([]Transaction, 0, len(transactions))
	for _, tx := range transactions {
		if Verify(tx) {
			valid = append(valid, tx)
		}
	}

	newBlock := Block{
		Index:        prev.Index + 1,
		Timestamp:    time.Now().UTC(),
		Transactions: valid,
		PreviousHash: prev.Hash,
		MerkleRoot:   ComputeMerkleRoot(valid),
	}
	newBlock.Hash = CalculateHash(newBlock)

	if !defaultValidatorSet.ReachConsensus(&newBlock) {
		fmt.Printf("WARNING: consensus failed for sequential block %d — block dropped\n", newBlock.Index)
		return
	}

	bc.Blocks = append(bc.Blocks, newBlock)
}
