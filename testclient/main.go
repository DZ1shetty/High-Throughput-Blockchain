package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const base = "http://localhost:8080"

// ── lightweight response types ────────────────────────────────────────────────

type statsResp struct {
	BlockCount        int     `json:"block_count"`
	TotalTransactions int     `json:"total_transactions"`
	UptimeSeconds     float64 `json:"uptime_seconds"`
	NumShards         int     `json:"num_shards"`
}

type txResp struct {
	Success bool `json:"success"`
}

type transaction struct {
	TxID      string  `json:"tx_id"`
	Sender    string  `json:"sender"`
	Receiver  string  `json:"receiver"`
	Amount    float64 `json:"amount"`
	Signature string  `json:"signature"`
}

type block struct {
	Index                int           `json:"index"`
	Transactions         []transaction `json:"transactions"`
	MerkleRoot           string        `json:"merkle_root"`
	ConsensusSignatures  []string      `json:"consensus_signatures"`
	Hash                 string        `json:"hash"`
}

// ── helpers ───────────────────────────────────────────────────────────────────

func getJSON(url string, dst any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, dst)
}

func postJSON(url string, payload any, dst any) (*http.Response, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if dst != nil {
		_ = json.Unmarshal(body, dst)
	}
	return resp, nil
}

// ── main ─────────────────────────────────────────────────────────────────────

func main() {
	passed := 0

	fmt.Println("========================================")
	fmt.Println("  High-Throughput Blockchain Test Suite")
	fmt.Println("========================================")

	// ── Test 1: server reachability ──────────────────────────────────────────
	fmt.Print("\nTest 1: GET /stats (server reachability) ... ")
	var statsBefore statsResp
	if err := getJSON(base+"/stats", &statsBefore); err != nil {
		fmt.Println("[FAIL] Server not running - start it with: go run .")
		os.Exit(1)
	}
	fmt.Println("[PASS] Server is running")
	passed++

	// ── Test 2: POST /transaction ─────────────────────────────────────────────
	fmt.Print("Test 2: POST /transaction              ... ")
	payload := map[string]any{
		"sender":   "TestUser",
		"receiver": "Kiro",
		"amount":   1,
	}
	var txResult txResp
	resp, err := postJSON(base+"/transaction", payload, &txResult)
	if err != nil || resp.StatusCode != http.StatusCreated || !txResult.Success {
		fmt.Printf("[FAIL] Transaction rejected (status %v, err: %v)\n", func() string {
			if resp != nil {
				return resp.Status
			}
			return "no response"
		}(), err)
	} else {
		fmt.Println("[PASS] Transaction accepted")
		passed++
	}

	// ── Test 3: transaction count increased ───────────────────────────────────
	fmt.Print("Test 3: GET /stats (tx count +1)       ... ")
	var statsAfter statsResp
	if err := getJSON(base+"/stats", &statsAfter); err != nil {
		fmt.Println("[FAIL] Could not reach /stats after transaction")
	} else if statsAfter.TotalTransactions != statsBefore.TotalTransactions+1 {
		fmt.Printf("[FAIL] Expected total_transactions=%d, got %d\n",
			statsBefore.TotalTransactions+1, statsAfter.TotalTransactions)
	} else {
		fmt.Println("[PASS] Transaction saved to chain")
		passed++
	}

	// ── Test 4: digital signatures ────────────────────────────────────────────
	fmt.Print("Test 4: GET /blocks (signature check)  ... ")
	var blocks []block
	if err := getJSON(base+"/blocks", &blocks); err != nil {
		fmt.Println("[FAIL] Could not reach /blocks")
	} else {
		// Walk blocks from newest to oldest looking for a signed transaction.
		found := false
		for i := len(blocks) - 1; i >= 0; i-- {
			for _, tx := range blocks[i].Transactions {
				if tx.Signature != "" {
					fmt.Println("[PASS] Digital signatures working")
					passed++
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			fmt.Println("[INFO] No signed transactions yet")
		}
	}

	// ── Test 5: consensus signatures ─────────────────────────────────────────
	fmt.Print("Test 5: GET /blocks (consensus check)  ... ")
	var blocksForConsensus []block
	if err := getJSON(base+"/blocks", &blocksForConsensus); err != nil {
		fmt.Println("[FAIL] Could not reach /blocks")
	} else if len(blocksForConsensus) == 0 {
		fmt.Println("[FAIL] No blocks in chain")
	} else {
		newest := blocksForConsensus[len(blocksForConsensus)-1]
		if len(newest.ConsensusSignatures) >= 3 {
			fmt.Printf("[PASS] Consensus achieved (2/3+ validators signed, got %d/4)\n",
				len(newest.ConsensusSignatures))
			passed++
		} else {
			fmt.Printf("[FAIL] No consensus signatures on newest block (got %d, need ≥3)\n",
				len(newest.ConsensusSignatures))
		}
	}

	// ── Summary ───────────────────────────────────────────────────────────────
	fmt.Println("\n========================================")
	fmt.Printf("  %d/5 TESTS PASSED\n", passed)
	fmt.Println("========================================")
}
