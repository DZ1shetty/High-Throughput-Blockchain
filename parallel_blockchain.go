package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ShardedBlockchain processes transactions across NumShards parallel shards
// and merges the resulting blocks into a single Blocks slice.
type ShardedBlockchain struct {
	NumShards int     `json:"num_shards"`
	Blocks    []Block `json:"blocks"`
}

// NewShardedBlockchain creates a ShardedBlockchain with a genesis block.
func NewShardedBlockchain(numShards int) *ShardedBlockchain {
	if numShards < 1 {
		numShards = 1
	}

	genesis := Block{
		Index:        0,
		Timestamp:    time.Now().UTC(),
		Transactions: []Transaction{},
		PreviousHash: "0",
	}
	genesis.Hash = calculateShardHash(genesis)

	return &ShardedBlockchain{
		NumShards: numShards,
		Blocks:    []Block{genesis},
	}
}

// calculateShardHash is a local SHA-256 helper (mirrors CalculateHash but kept
// self-contained so this file has no dependency on blockchain.go).
// MerkleRoot is included so the hash covers all transaction data.
func calculateShardHash(block Block) string {
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
		data = []byte(fmt.Sprintf("%d%s%s%s%v",
			block.Index, block.Timestamp, block.PreviousHash, block.MerkleRoot, block.Transactions))
	}

	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum)
}

// AddBlockParallel distributes transactions across NumShards goroutines.
// Shard k receives transactions at indices k, k+NumShards, k+2*NumShards, …
// Each shard builds and hashes its own partial Block, runs consensus, and
// sends its block only if ≥2/3 validators signed it.
// Blocks that fail consensus are dropped with a warning.
func (sb *ShardedBlockchain) AddBlockParallel(transactions []Transaction) {
	prev := sb.Blocks[len(sb.Blocks)-1]
	baseIndex := prev.Index + 1
	now := time.Now().UTC()

	type result struct {
		shardID int
		block   Block
		ok      bool
	}
	ch := make(chan result, sb.NumShards)

	var wg sync.WaitGroup

	for s := 0; s < sb.NumShards; s++ {
		wg.Add(1)
		go func(shardID int) {
			defer wg.Done()

			// Collect this shard's transactions (interleaved assignment).
			// Skip any transaction whose signature is invalid.
			var shardTxs []Transaction
			for i := shardID; i < len(transactions); i += sb.NumShards {
				if Verify(transactions[i]) {
					shardTxs = append(shardTxs, transactions[i])
				}
			}

			b := Block{
				Index:        baseIndex + shardID,
				Timestamp:    now,
				Transactions: shardTxs,
				PreviousHash: prev.Hash,
				MerkleRoot:   ComputeMerkleRoot(shardTxs),
			}
			b.Hash = calculateShardHash(b)

			if !defaultValidatorSet.ReachConsensus(&b) {
				fmt.Printf("WARNING: consensus failed for shard block %d — block dropped\n", b.Index)
				ch <- result{shardID: shardID, ok: false}
				return
			}

			ch <- result{shardID: shardID, block: b, ok: true}
		}(s)
	}

	// Close the channel once all goroutines finish.
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Drain results, preserving shard order for successful blocks.
	ordered := make([]result, sb.NumShards)
	for res := range ch {
		ordered[res.shardID] = res
	}

	for _, res := range ordered {
		if res.ok {
			sb.Blocks = append(sb.Blocks, res.block)
		}
	}
}
