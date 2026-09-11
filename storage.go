package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// chainFile is the on-disk representation of a ShardedBlockchain.
type chainFile struct {
	NumShards int     `json:"num_shards"`
	Blocks    []Block `json:"blocks"`
}

// SaveChain serialises bc to a JSON file at filename atomically:
// it writes to a temp file first, then renames, so a crash mid-write
// never leaves a corrupt file behind.
func SaveChain(bc *ShardedBlockchain, filename string) error {
	if bc == nil {
		return errors.New("SaveChain: blockchain is nil")
	}

	payload := chainFile{
		NumShards: bc.NumShards,
		Blocks:    bc.Blocks,
	}

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("SaveChain: marshal failed: %w", err)
	}

	// Write to a temp file in the same directory so the rename is atomic.
	tmp := filename + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("SaveChain: write temp file failed: %w", err)
	}

	if err := os.Rename(tmp, filename); err != nil {
		// Best-effort cleanup of the temp file on failure.
		_ = os.Remove(tmp)
		return fmt.Errorf("SaveChain: rename failed: %w", err)
	}

	return nil
}

// LoadChain reads filename and reconstructs a *ShardedBlockchain.
// Returns an error if the file cannot be read or is malformed.
// If the file does not exist, the caller should create a fresh chain instead.
func LoadChain(filename string) (*ShardedBlockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("LoadChain: read failed: %w", err)
	}

	var payload chainFile
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("LoadChain: unmarshal failed: %w", err)
	}

	if payload.NumShards < 1 {
		return nil, errors.New("LoadChain: invalid num_shards in file")
	}
	if len(payload.Blocks) == 0 {
		return nil, errors.New("LoadChain: chain has no blocks")
	}

	return &ShardedBlockchain{
		NumShards: payload.NumShards,
		Blocks:    payload.Blocks,
	}, nil
}
