package main

import (
	"errors"
	"log"
	"os"
)

const chainFilename = "blockchain.json"

func main() {
	bc, err := LoadChain(chainFilename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("No existing chain found — starting fresh (4 shards)")
		} else {
			log.Printf("Warning: could not load chain from %q: %v — starting fresh", chainFilename, err)
		}
		bc = NewShardedBlockchain(4)
	} else {
		log.Printf("Loaded chain from %q: %d blocks, %d shards",
			chainFilename, len(bc.Blocks), bc.NumShards)
	}

	wallet, err := NewWallet()
	if err != nil {
		log.Fatalf("Failed to create server wallet: %v", err)
	}
	log.Printf("Server wallet ready (public key: %s...)", wallet.PublicKeyHex[:16])

	StartServer(bc, wallet)
}
