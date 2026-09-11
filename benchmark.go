package main

import (
	"fmt"
	"time"
)

// BenchmarkResult holds the outcome of a single benchmark run.
type BenchmarkResult struct {
	TxCount             int     `json:"tx_count"`
	NumShards           int     `json:"num_shards"`
	SequentialTimeMs    float64 `json:"sequential_time_ms"`
	SequentialTPS       float64 `json:"sequential_tps"`
	ParallelTimeMs      float64 `json:"parallel_time_ms"`
	ParallelTPS         float64 `json:"parallel_tps"`
	SpeedupPercent      float64 `json:"speedup_percent"`
	SequentialBlocks    int     `json:"sequential_blocks"`
	ParallelBlocks      int     `json:"parallel_blocks"`
}

// runBenchmarkResult executes the benchmark and returns structured results.
func runBenchmarkResult() BenchmarkResult {
	const txCount = 10_000
	const numShards = 4

	w, err := NewWallet()
	if err != nil {
		panic("runBenchmarkResult: failed to create wallet: " + err.Error())
	}

	transactions := make([]Transaction, 0, txCount)
	for i := 0; i < txCount; i++ {
		tx := NewTransaction(
			fmt.Sprintf("sender_%d", i),
			fmt.Sprintf("receiver_%d", i),
			float64(i+1)*0.01,
		)
		if err := w.Sign(&tx); err != nil {
			panic("runBenchmarkResult: sign failed: " + err.Error())
		}
		transactions = append(transactions, tx)
	}

	// Sequential
	seqChain := NewBlockchain()
	seqStart := time.Now()
	seqChain.AddBlock(transactions)
	seqElapsed := time.Since(seqStart)
	seqTPS := float64(txCount) / seqElapsed.Seconds()

	// Parallel
	parChain := NewShardedBlockchain(numShards)
	parStart := time.Now()
	parChain.AddBlockParallel(transactions)
	parElapsed := time.Since(parStart)
	parTPS := float64(txCount) / parElapsed.Seconds()

	speedup := ((parTPS - seqTPS) / seqTPS) * 100

	return BenchmarkResult{
		TxCount:          txCount,
		NumShards:        numShards,
		SequentialTimeMs: float64(seqElapsed.Microseconds()) / 1000.0,
		SequentialTPS:    seqTPS,
		ParallelTimeMs:   float64(parElapsed.Microseconds()) / 1000.0,
		ParallelTPS:      parTPS,
		SpeedupPercent:   speedup,
		SequentialBlocks: len(seqChain.Blocks),
		ParallelBlocks:   len(parChain.Blocks),
	}
}

// RunBenchmark prints a human-readable benchmark comparison to stdout.
func RunBenchmark() {
	r := runBenchmarkResult()

	fmt.Printf("Generating %d test transactions...\n\n", r.TxCount)
	fmt.Println("========================================================")
	fmt.Println("               BENCHMARK RESULTS")
	fmt.Println("========================================================")
	fmt.Printf("%-28s %12s %15s\n", "Approach", "Time (ms)", "TPS")
	fmt.Println("--------------------------------------------------------")
	fmt.Printf("%-28s %12.3f %15.2f\n", "Sequential", r.SequentialTimeMs, r.SequentialTPS)
	fmt.Printf("%-28s %12.3f %15.2f\n",
		fmt.Sprintf("Parallel (%d shards)", r.NumShards),
		r.ParallelTimeMs, r.ParallelTPS,
	)
	fmt.Println("--------------------------------------------------------")
	fmt.Printf("Speedup (parallel vs sequential): %+.2f%%\n", r.SpeedupPercent)
	fmt.Println("========================================================")
	fmt.Printf("\nSequential blocks : %d\n", r.SequentialBlocks)
	fmt.Printf("Parallel blocks   : %d (1 genesis + %d shards)\n", r.ParallelBlocks, r.NumShards)
}

// ShardScalingResult holds the result for a single shard count in the scaling test.
type ShardScalingResult struct {
	NumShards int     `json:"num_shards"`
	TimeMs    float64 `json:"time_ms"`
	TPS       float64 `json:"tps"`
}

// runScalingResult runs AddBlockParallel for each shard count in shardCounts
// against the same 10,000 transactions and returns structured results.
func runScalingResult() []ShardScalingResult {
	const txCount = 10_000
	shardCounts := []int{1, 2, 4, 8, 16}

	w, err := NewWallet()
	if err != nil {
		panic("runScalingResult: failed to create wallet: " + err.Error())
	}

	// Generate transactions once; reuse across all shard runs.
	transactions := make([]Transaction, 0, txCount)
	for i := 0; i < txCount; i++ {
		tx := NewTransaction(
			fmt.Sprintf("sender_%d", i),
			fmt.Sprintf("receiver_%d", i),
			float64(i+1)*0.01,
		)
		if err := w.Sign(&tx); err != nil {
			panic("runScalingResult: sign failed: " + err.Error())
		}
		transactions = append(transactions, tx)
	}

	results := make([]ShardScalingResult, 0, len(shardCounts))
	for _, n := range shardCounts {
		chain := NewShardedBlockchain(n)
		start := time.Now()
		chain.AddBlockParallel(transactions)
		elapsed := time.Since(start)
		tps := float64(txCount) / elapsed.Seconds()

		results = append(results, ShardScalingResult{
			NumShards: n,
			TimeMs:    float64(elapsed.Microseconds()) / 1000.0,
			TPS:       tps,
		})
	}
	return results
}

// RunScalingBenchmark prints a shard-scaling table to stdout.
func RunScalingBenchmark() {
	const txCount = 10_000
	fmt.Printf("Scaling benchmark — %d transactions, shard counts: 1 2 4 8 16\n\n", txCount)
	fmt.Println("========================================================")
	fmt.Println("            SHARD SCALING RESULTS")
	fmt.Println("========================================================")
	fmt.Printf("%-12s %14s %18s\n", "Num Shards", "Time (ms)", "TPS")
	fmt.Println("--------------------------------------------------------")

	for _, r := range runScalingResult() {
		fmt.Printf("%-12d %14.3f %18.2f\n", r.NumShards, r.TimeMs, r.TPS)
	}

	fmt.Println("========================================================")
}
