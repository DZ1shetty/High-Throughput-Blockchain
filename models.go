package main

import (
	"time"

	"github.com/google/uuid"
)

// Transaction represents a single transfer of value between two parties.
type Transaction struct {
	TxID      string    `json:"tx_id"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Amount    float64   `json:"amount"`
	Timestamp time.Time `json:"timestamp"`
	PublicKey string    `json:"public_key"`
	Signature string    `json:"signature"`
}

// Block represents a single block in the blockchain.
type Block struct {
	Index                int           `json:"index"`
	Timestamp            time.Time     `json:"timestamp"`
	Transactions         []Transaction `json:"transactions"`
	PreviousHash         string        `json:"previous_hash"`
	MerkleRoot           string        `json:"merkle_root"`
	ConsensusSignatures  []string      `json:"consensus_signatures"`
	Hash                 string        `json:"hash"`
}

// NewTransaction creates a new Transaction with a randomly generated UUID
// for the TxID and the current UTC time as the Timestamp.
func NewTransaction(sender, receiver string, amount float64) Transaction {
	return Transaction{
		TxID:      uuid.New().String(),
		Sender:    sender,
		Receiver:  receiver,
		Amount:    amount,
		Timestamp: time.Now().UTC(),
	}
}
