# Distributed In-Memory Counter in Go

This project implements a distributed in-memory counter system that achieves eventual consistency through a custom-built service discovery mechanism. Nodes can join a cluster, discover each other, and propagate counter increments in a fault-tolerant manner.

## Design Decisions

### Service Discovery

The service discovery is implemented using a simple, serverless gossip protocol.

- **Joining**: A new node announces its presence to a predefined list of seed peers.
- **Peer Synchronization**: When a node joins, it receives a list of known peers from the seed node it contacted. It merges this list with its own, quickly gaining a view of the cluster.
- **Health Checks**: Nodes periodically send lightweight heartbeat messages to their known peers. If a peer is unresponsive for a configurable duration, it is removed from the active list. This makes the cluster resilient to node failures.

This approach was chosen for its simplicity and decentralization, avoiding a single point of failure.

### Eventual Consistency & Deduplication

The system is designed for eventual consistency.

- **Local Reads**: `GET /count` returns the node's local value of the counter, which may not be globally consistent at the exact moment of the request. Over time, all nodes will converge to the same value.
- **Idempotent Increments**: Every increment operation is assigned a unique UUID. When an increment is propagated, nodes check if they have already processed this UUID. If so, they ignore the request. This prevents duplicate counting, which is critical during network partitions or message retries.
- **Failure Handling**: If propagating an increment to a peer fails, the operation is retried with an exponential backoff strategy. This handles transient network issues gracefully.

## How to Run

1.  **Clone the repository and navigate to the project directory.**
2.  **Open multiple terminal windows.**

**Terminal 1: Start the first node (seed node)**

```bash
go run ./cmd/server --port=8080
```

**Terminal 2: Start a second node and have it join the first**

```bash
go run ./cmd/server --port=8081 --peers=localhost:8080
```

**Terminal 3: Start a third node and have it join the cluster via any known node**

```bash
go run ./cmd/server --port=8082 --peers=localhost:8081
```

## API Usage

**Increment the counter (can be sent to any node):**

```bash
curl -X POST http://localhost:8080/increment
```

**Get the current count from any node:**

```bash
curl http://localhost:8080/count
```

**Wait a moment for propagation and check another node**

```bash
curl http://localhost:8082/count
```

## How to Test

**Run all unit tests, including race condition checks, and generate a coverage report.**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
