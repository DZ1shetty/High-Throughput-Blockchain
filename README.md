# High-Throughput Blockchain Architecture

A high-performance blockchain implementation in Go demonstrating parallel transaction processing through sharding, ed25519 digital signatures, PBFT-style validator consensus, Merkle tree integrity, and atomic disk persistence — all exposed through a lightweight REST API.

---

## Features

- **Parallel sharding** — transactions are distributed across N shards and processed simultaneously using Go goroutines and channels
- **Sequential vs parallel comparison** — built-in benchmark measures TPS for both approaches on 10,000 transactions
- **Shard scaling benchmark** — tests 1, 2, 4, 8, and 16 shards in one pass to show how TPS scales with parallelism
- **Merkle tree** — each block computes a SHA-256 Merkle root over its transaction IDs; the root is included in the block hash
- **SHA-256 hashing** — every block is cryptographically hashed using `crypto/sha256`; the hash covers index, timestamp, transactions, previous hash, and Merkle root
- **ed25519 digital signatures** — every transaction is signed with an ed25519 key pair; invalid signatures are rejected before a transaction enters a block; legacy unsigned transactions are accepted for backward compatibility
- **PBFT-style consensus** — 4 independent validators each verify the block hash and Merkle root, then sign the block hash in parallel; a block is only committed if ≥ ⌈2/3⌉ validators sign (Byzantine fault tolerance)
- **UUID transaction IDs** — each transaction receives a globally unique ID via `github.com/google/uuid`
- **Disk persistence** — the chain is saved atomically to `blockchain.json` after every block addition using a write-then-rename strategy; the file is reloaded automatically on startup
- **REST API** — submit transactions, query chain stats, inspect blocks with Merkle roots and consensus signatures, and trigger benchmarks over HTTP
- **Automated test suite** — 5-test client in `testclient/` verifies server health, transaction submission, chain state, digital signatures, and consensus
- **Thread-safe** — concurrent API requests are protected with `sync.Mutex` and atomic in-flight flags

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                        API Layer                        │
│   POST /transaction   GET /stats   GET /blocks  ...     │
└───────────────────────────┬─────────────────────────────┘
                            │  signed Transaction
                            ▼
┌─────────────────────────────────────────────────────────┐
│                     Shard Router                        │
│   Tx[0,4,8…] → Shard 0   Tx[1,5,9…] → Shard 1  …      │
└──────────┬──────────────────┬──────────────┬────────────┘
           │ goroutine        │ goroutine    │ goroutine
           ▼                  ▼              ▼
┌──────────────────┐  ┌───────────────┐  ┌──────────────┐
│   Shard Block 0  │  │ Shard Block 1 │  │ Shard Block N│
│ verify sigs      │  │ verify sigs   │  │ verify sigs  │
│ compute Merkle   │  │ compute Merkle│  │ compute Merkle│
│ SHA-256 hash     │  │ SHA-256 hash  │  │ SHA-256 hash │
└────────┬─────────┘  └──────┬────────┘  └──────┬───────┘
         │                   │                  │
         └───────────────────┴──────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────┐
│              Validator Consensus  (PBFT-style)          │
│                                                         │
│  Validator 0 ──┐                                        │
│  Validator 1 ──┤──▶  check hash + Merkle root           │
│  Validator 2 ──┤     sign block hash (ed25519)          │
│  Validator 3 ──┘     require ≥ 3/4 signatures           │
│                      (⌈2/3⌉ Byzantine quorum)           │
└───────────────────────────┬─────────────────────────────┘
                            │  committed block
                            ▼
┌─────────────────────────────────────────────────────────┐
│                    Block Storage                        │
│   Blocks[]  ──▶  SaveChain()  ──▶  blockchain.json      │
│   (atomic write: .tmp → rename)                        │
└─────────────────────────────────────────────────────────┘
```

---

## Benchmark Results

### Sequential vs Parallel (4 shards, 10,000 transactions)

| Approach            | Time Taken | TPS      | Blocks |
|---------------------|------------|----------|--------|
| Sequential          | ~41.15 ms  | ~243,000 | 2      |
| Parallel (4 shards) | ~19.49 ms  | ~513,000 | 5      |

**Speedup: ~2.1× faster with parallel sharding**

### Shard Scaling (10,000 transactions, signed with ed25519)

| Num Shards | Time (ms) | TPS       |
|------------|-----------|-----------|
| 1          | ~78.1 ms  | ~128,000  |
| 2          | ~41.0 ms  | ~244,000  |
| 4          | ~19.5 ms  | ~513,000  |
| 8          | ~15.4 ms  | ~650,000  |
| 16         | ~12.2 ms  | ~818,000  |

TPS scales with shard count, tapering at higher values as goroutine scheduling overhead grows relative to per-shard work. Benchmarks include ed25519 signing and verification, Merkle root computation, SHA-256 hashing, and PBFT consensus.

> Results vary by hardware. Run `GET /benchmark` or `GET /scaling` to measure on your machine.

---

## Test Suite

Run the automated test suite against a live server:

```bash
# Terminal 1
go run .

# Terminal 2
go run ./testclient/main.go
```

**Expected output:**

```
========================================
  High-Throughput Blockchain Test Suite
========================================
Test 1: GET /stats (server reachability) ... [PASS] Server is running
Test 2: POST /transaction              ... [PASS] Transaction accepted
Test 3: GET /stats (tx count +1)       ... [PASS] Transaction saved to chain
Test 4: GET /blocks (signature check)  ... [PASS] Digital signatures working
Test 5: GET /blocks (consensus check)  ... [PASS] Consensus achieved (2/3+ validators signed, got 4/4)
========================================
  5/5 TESTS PASSED
========================================
```

---

## Installation

**Prerequisites:** [Go 1.21+](https://go.dev/dl/)

```bash
# 1. Clone the repository
git clone https://github.com/your-username/high-throughput-blockchain.git
cd high-throughput-blockchain

# 2. Install dependencies
go mod tidy

# 3. Build
go build -o blockchain-server .

# 4. Start the server
./blockchain-server
```

The server starts on `http://localhost:8080`. The chain is persisted to `blockchain.json` and reloaded on the next startup.

---

## API Documentation

| Method | Endpoint       | Description                                                  |
|--------|----------------|--------------------------------------------------------------|
| POST   | `/transaction` | Submit a transaction; server signs, validates, and commits   |
| GET    | `/stats`       | Block count, total transactions, uptime, shard count         |
| GET    | `/blocks`      | Full chain as JSON — includes MerkleRoot and consensus sigs  |
| GET    | `/benchmark`   | Sequential vs 4-shard parallel benchmark (10,000 tx), JSON  |
| GET    | `/scaling`     | Shard scaling benchmark (1,2,4,8,16 shards), JSON           |

### POST /transaction

```bash
curl -X POST http://localhost:8080/transaction \
  -H "Content-Type: application/json" \
  -d '{"sender": "alice", "receiver": "bob", "amount": 42.5}'
```

**Response:**

```json
{
  "success": true,
  "transaction": {
    "tx_id": "3eb621c6-225e-4c67-822f-12c0eb6bb7d6",
    "sender": "alice",
    "receiver": "bob",
    "amount": 42.5,
    "timestamp": "2026-09-11T10:55:52.956Z",
    "public_key": "8585acbcfb213d05c0c9fb9ba23d9f826b0cdde32693fa5fd599d1f7f3b8156a",
    "signature": "287b95ba5d09ace1aaf8836f103338b17600502f6c52faef..."
  },
  "block_count": 5
}
```

### GET /stats

```bash
curl http://localhost:8080/stats
```

```json
{
  "block_count": 5,
  "total_transactions": 1,
  "uptime_seconds": 14.3,
  "num_shards": 4
}
```

### GET /blocks

Returns the full chain. Each block includes its Merkle root, consensus signatures, and all transactions with their ed25519 signatures.

```bash
curl http://localhost:8080/blocks
```

**Response (single block excerpt):**

```json
{
  "index": 1,
  "timestamp": "2026-09-11T10:55:52.956Z",
  "transactions": [
    {
      "tx_id": "3eb621c6-225e-4c67-822f-12c0eb6bb7d6",
      "sender": "alice",
      "receiver": "bob",
      "amount": 42.5,
      "timestamp": "2026-09-11T10:55:52.956Z",
      "public_key": "8585acbcfb213d05c0c9fb9ba23d9f826b0cdde32693fa5fd599d1f7f3b8156a",
      "signature": "287b95ba5d09ace1aaf8836f103338b176..."
    }
  ],
  "previous_hash": "c80cd5d570ce37274583e4ff10500cacd94cb8ba2b45e84da89eccb83f96fc17",
  "merkle_root": "5b72a3aa3369250b17d19928f77df2d6c19e749b9e00ee02426a7028b783aafd",
  "consensus_signatures": [
    "09248b29fcbbcf7a86862cc8657b9ca3ea7c5ccaabcf709e25ab14c8f0096a9...",
    "4a5018b325935328c2e896ef1d857af0eeac0f31cc0f12379d9f3d98de848e0...",
    "4fcf7d379d5abe9168b4fdb2634f67ad4f66f90bef9dd8db0e61d278558781303...",
    "ff890aafa9d5f85e37537065051c1f2c1c0345f8af2fb4a5bbabd01d2bb6081..."
  ],
  "hash": "9a8ae62d190cb856b55c357a8c3910b65b65892b929a7c6d6e03fcbdab08ffe0"
}
```

### GET /benchmark

```bash
curl http://localhost:8080/benchmark
```

```json
{
  "tx_count": 10000,
  "num_shards": 4,
  "sequential_time_ms": 41.15,
  "sequential_tps": 243013.5,
  "parallel_time_ms": 19.49,
  "parallel_tps": 513084.2,
  "speedup_percent": 111.13,
  "sequential_blocks": 2,
  "parallel_blocks": 5
}
```

### GET /scaling

```bash
curl http://localhost:8080/scaling
```

```json
[
  { "num_shards": 1,  "time_ms": 78.1,  "tps": 128054.3 },
  { "num_shards": 2,  "time_ms": 41.0,  "tps": 243902.4 },
  { "num_shards": 4,  "time_ms": 19.5,  "tps": 512820.5 },
  { "num_shards": 8,  "time_ms": 15.4,  "tps": 649350.6 },
  { "num_shards": 16, "time_ms": 12.2,  "tps": 818032.8 }
]
```

---

## Project Structure

```
high-throughput-blockchain/
├── main.go                  # Entry point — loads or creates chain, creates wallet, starts server
├── models.go                # Transaction and Block structs (with PublicKey, Signature, MerkleRoot, ConsensusSignatures)
├── blockchain.go            # Sequential Blockchain, NewBlockchain(), AddBlock(), CalculateHash()
├── parallel_blockchain.go   # ShardedBlockchain, NewShardedBlockchain(), AddBlockParallel()
├── merkle.go                # ComputeMerkleRoot() — SHA-256 Merkle tree over transaction IDs
├── wallet.go                # Wallet, NewWallet(), Sign(), Verify() — ed25519 key management
├── consensus.go             # ValidatorSet, NewValidatorSet(), ReachConsensus() — PBFT-style voting
├── benchmark.go             # RunBenchmark(), RunScalingBenchmark(), JSON-returning variants
├── api.go                   # HTTP handlers for all endpoints; StartServer()
├── storage.go               # SaveChain() and LoadChain() — atomic JSON persistence
├── testclient/
│   └── main.go              # 5-test automated client — run against a live server
├── go.mod                   # Module definition and Go version
└── go.sum                   # Dependency checksums
```

---

## How It All Fits Together

```
NewTransaction()  ──▶  Sign(tx)  ──▶  POST /transaction
                                            │
                                            ▼
                                   AddBlockParallel()
                                            │
                              ┌─────────────┼─────────────┐
                         Shard 0        Shard 1 …      Shard N
                         verify         verify          verify
                         Merkle         Merkle          Merkle
                         SHA-256        SHA-256         SHA-256
                              └─────────────┼─────────────┘
                                            │
                                   ReachConsensus()
                                   4 validators, ≥3 sign
                                            │
                                     committed ✓
                                            │
                                      SaveChain()
                                   blockchain.json
```

### Merkle Tree

```
Transactions:  [TxA,         TxB,         TxC,         TxD        ]
Leaves:        [H(TxA),      H(TxB),      H(TxC),      H(TxD)     ]
Level 1:       [H(L0+L1),                 H(L2+L3)                 ]
Root:          [H(N0+N1)                                           ]
```

If a layer has an odd number of nodes, the last node is duplicated before hashing.

### PBFT Consensus

```
Block hash computed
        │
        ├──▶ Validator 0: check hash + Merkle → sign ──┐
        ├──▶ Validator 1: check hash + Merkle → sign ──┤ parallel goroutines
        ├──▶ Validator 2: check hash + Merkle → sign ──┤
        └──▶ Validator 3: check hash + Merkle → sign ──┘
                                                        │
                              count signatures ≥ ⌈2/3⌉ = 3 ?
                                        │
                              YES ──▶  commit block
                              NO  ──▶  drop block + log warning
```

---

## License

MIT
