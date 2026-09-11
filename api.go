package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// server bundles the shared state accessible to all HTTP handlers.
type server struct {
	bc        *ShardedBlockchain
	wallet    *Wallet
	startTime time.Time
	mu        sync.Mutex // guards bc mutations
}

// ── response helpers ────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON encode error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ── /transaction ─────────────────────────────────────────────────────────────

type txRequest struct {
	Sender   string  `json:"sender"`
	Receiver string  `json:"receiver"`
	Amount   float64 `json:"amount"`
}

type txResponse struct {
	Success     bool        `json:"success"`
	Transaction Transaction `json:"transaction"`
	BlockCount  int         `json:"block_count"`
}

func (s *server) handleTransaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is allowed")
		return
	}

	var req txRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if req.Sender == "" || req.Receiver == "" {
		writeError(w, http.StatusBadRequest, "sender and receiver must not be empty")
		return
	}
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount must be greater than zero")
		return
	}

	tx := NewTransaction(req.Sender, req.Receiver, req.Amount)
	if err := s.wallet.Sign(&tx); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to sign transaction: "+err.Error())
		return
	}

	s.mu.Lock()
	s.bc.AddBlockParallel([]Transaction{tx})
	blockCount := len(s.bc.Blocks)
	saveErr := SaveChain(s.bc, chainFilename)
	s.mu.Unlock()

	if saveErr != nil {
		// The transaction was added to memory; warn but don't fail the request.
		log.Printf("WARNING: transaction accepted but chain save failed: %v", saveErr)
	}

	writeJSON(w, http.StatusCreated, txResponse{
		Success:     true,
		Transaction: tx,
		BlockCount:  blockCount,
	})
}

// ── /stats ───────────────────────────────────────────────────────────────────

type statsResponse struct {
	BlockCount        int     `json:"block_count"`
	TotalTransactions int     `json:"total_transactions"`
	UptimeSeconds     float64 `json:"uptime_seconds"`
	NumShards         int     `json:"num_shards"`
}

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "only GET is allowed")
		return
	}

	s.mu.Lock()
	blockCount := len(s.bc.Blocks)
	totalTx := 0
	for _, b := range s.bc.Blocks {
		totalTx += len(b.Transactions)
	}
	numShards := s.bc.NumShards
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, statsResponse{
		BlockCount:        blockCount,
		TotalTransactions: totalTx,
		UptimeSeconds:     time.Since(s.startTime).Seconds(),
		NumShards:         numShards,
	})
}

// ── /benchmark ───────────────────────────────────────────────────────────────

// benchmarkInFlight prevents concurrent benchmark runs that would saturate CPU.
var benchmarkInFlight atomic.Bool

func (s *server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "only GET is allowed")
		return
	}

	if !benchmarkInFlight.CompareAndSwap(false, true) {
		writeError(w, http.StatusTooManyRequests, "a benchmark is already running")
		return
	}
	defer benchmarkInFlight.Store(false)

	result := runBenchmarkResult()
	writeJSON(w, http.StatusOK, result)
}

// ── /blocks ──────────────────────────────────────────────────────────────────

func (s *server) handleBlocks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "only GET is allowed")
		return
	}

	s.mu.Lock()
	// Copy the slice so we release the lock before encoding.
	blocks := make([]Block, len(s.bc.Blocks))
	copy(blocks, s.bc.Blocks)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, blocks)
}

// ── /scaling ─────────────────────────────────────────────────────────────────

// scalingInFlight prevents concurrent scaling runs from stacking up.
var scalingInFlight atomic.Bool

func (s *server) handleScaling(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "only GET is allowed")
		return
	}

	if !scalingInFlight.CompareAndSwap(false, true) {
		writeError(w, http.StatusTooManyRequests, "a scaling benchmark is already running")
		return
	}
	defer scalingInFlight.Store(false)

	results := runScalingResult()
	writeJSON(w, http.StatusOK, results)
}

// ── StartServer ──────────────────────────────────────────────────────────────

// StartServer registers all routes and blocks while serving on :8080.
func StartServer(bc *ShardedBlockchain, wallet *Wallet) {
	s := &server{
		bc:        bc,
		wallet:    wallet,
		startTime: time.Now(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/transaction", s.handleTransaction)
	mux.HandleFunc("/stats", s.handleStats)
	mux.HandleFunc("/benchmark", s.handleBenchmark)
	mux.HandleFunc("/scaling", s.handleScaling)
	mux.HandleFunc("/blocks", s.handleBlocks)

	addr := ":8080"
	fmt.Printf("Server listening on http://localhost%s\n", addr)
	fmt.Println("  POST /transaction  — submit a transaction")
	fmt.Println("  GET  /stats        — chain statistics")
	fmt.Println("  GET  /benchmark    — run sequential vs parallel benchmark")
	fmt.Println("  GET  /scaling      — run shard scaling benchmark (1,2,4,8,16 shards)")
	fmt.Println("  GET  /blocks       — list all blocks with MerkleRoot and hashes")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
