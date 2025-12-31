# Phase 7: Distributed Systems - Learning Guide

This document covers distributed systems fundamentals: consensus algorithms (Raft), streaming communication patterns, leader election, and log replication. Phase 7 brings everything together to build systems that work across multiple nodes.

---

## Prerequisites

Before starting Phase 7, ensure you've completed:
- [x] Phase 1: Foundations (interfaces, error handling, testing)
- [x] Phase 2: Concurrency (goroutines, channels, sync primitives)
- [x] Phase 3: Generics (type parameters, generic data structures)
- [x] Phase 4: Performance (profiling, benchmarks, memory pooling)
- [x] Phase 5: Systems Programming (unsafe, mmap, CGO)
- [x] Phase 6: Static Analysis (AST parsing, linting, code generation)

---

## Why Distributed Systems?

Modern applications rarely run on a single machine. Distributed systems enable:
1. **Fault Tolerance:** Survive node failures
2. **Scalability:** Handle more load by adding nodes
3. **Availability:** Serve requests even during partial outages
4. **Data Locality:** Process data closer to users

Go excels at distributed systems due to its concurrency primitives, efficient networking, and clean error handling.

---

## 1. Consensus and Raft

**Files:** `pkg/distributed/raft.go`, `pkg/distributed/raft_test.go`

### Concept

Consensus algorithms ensure multiple nodes agree on the same data, even when some nodes fail. Raft is designed for understandability while providing strong consistency guarantees.

### What You'll Learn

```go
// LEARN: Raft node states
// Every node is in exactly one of these states

type NodeState int

const (
    Follower  NodeState = iota  // Default state, accepts leader's commands
    Candidate                    // Seeking votes to become leader
    Leader                       // Handles all client requests
)

// LEARN: Why these states?
// - Follower: Most nodes most of the time (efficiency)
// - Candidate: Transient state during election
// - Leader: Single coordinator (simplicity)
```

### Log Entries and Terms

```go
// LEARN: Every operation is a log entry
// Entries are ordered and replicated to all nodes

type LogEntry struct {
    Index   uint64      // Position in the log
    Term    uint64      // Election term when entry was created
    Command interface{} // The actual operation
}

// LEARN: Terms are logical clocks
// - New election starts new term
// - Term numbers only increase
// - Higher term = more recent leader
```

### Leader Election

```go
// LEARN: Election triggers
// Follower starts election when:
// 1. Election timeout expires (no heartbeat from leader)
// 2. Leader disconnects or fails

func (n *RaftNode) startElection() {
    n.mu.Lock()
    n.currentTerm++       // Increment term
    n.state = Candidate   // Become candidate
    n.votedFor = n.id     // Vote for self
    n.mu.Unlock()
    
    votes := 1  // Already have own vote
    
    // LEARN: Request votes from all peers in parallel
    for _, peer := range n.peers {
        go func(p Peer) {
            reply, err := p.RequestVote(RequestVoteArgs{
                Term:         n.currentTerm,
                CandidateID:  n.id,
                LastLogIndex: n.log.LastIndex(),
                LastLogTerm:  n.log.LastTerm(),
            })
            
            if err != nil || !reply.VoteGranted {
                return
            }
            
            // Count vote
            if atomic.AddInt32(&votes, 1) > len(n.peers)/2 {
                n.becomeLeader()
            }
        }(peer)
    }
}
```

### Log Replication

```go
// LEARN: Leader replicates entries to followers
// Once majority acknowledges, entry is "committed"

func (n *RaftNode) appendEntries(entry LogEntry) error {
    // Step 1: Append to own log
    n.log.Append(entry)
    
    // Step 2: Replicate to followers
    acks := 1  // Self already has it
    
    for _, peer := range n.peers {
        go func(p Peer) {
            reply, err := p.AppendEntries(AppendEntriesArgs{
                Term:         n.currentTerm,
                LeaderID:     n.id,
                PrevLogIndex: entry.Index - 1,
                PrevLogTerm:  n.log.Term(entry.Index - 1),
                Entries:      []LogEntry{entry},
                LeaderCommit: n.commitIndex,
            })
            
            if err != nil || !reply.Success {
                return
            }
            
            // LEARN: Majority replication = committed
            if atomic.AddInt32(&acks, 1) > len(n.peers)/2 {
                n.commitIndex = entry.Index
                n.applyCommitted()
            }
        }(peer)
    }
    
    return nil
}
```

### Safety Guarantees

```go
// LEARN: Raft provides these guarantees:

// 1. Election Safety: At most one leader per term
//    - Each node votes at most once per term
//    - Candidate needs majority to win

// 2. Leader Append-Only: Leaders never overwrite or delete entries
//    - Only append new entries
//    - Followers may truncate to match leader

// 3. Log Matching: If two logs have entry with same index and term,
//    all preceding entries are identical
//    - AppendEntries includes prev log info for consistency check

// 4. Leader Completeness: Committed entries appear in all future leaders
//    - Voters only vote for candidates with up-to-date logs

// 5. State Machine Safety: Once entry applied, no other server
//    applies different command at same index
```

### Key Principles

- **Single Leader:** Simplifies reasoning about system state
- **Majority Quorum:** Tolerates (N-1)/2 failures
- **Persistent State:** votedFor, currentTerm, log survive restarts
- **Idempotent:** Duplicate requests don't cause issues

---

## 2. Streaming Communication

**Files:** `pkg/distributed/stream.go`, `pkg/distributed/stream_test.go`

### Concept

Streaming enables efficient transfer of large datasets or continuous updates. Unlike request-response, streams keep connections open for multiple messages.

### What You'll Learn

```go
// LEARN: Stream types
// Inspired by gRPC but framework-agnostic

type StreamType int

const (
    UnaryStream       StreamType = iota // Single request, single response
    ServerStream                        // Single request, multiple responses
    ClientStream                        // Multiple requests, single response
    BidirectionalStream                 // Multiple requests and responses
)

// LEARN: Each type serves different use cases:
// - Unary: Traditional RPC, simple operations
// - ServerStream: Subscribe to updates, download chunks
// - ClientStream: Upload file chunks, batch operations
// - Bidirectional: Chat, real-time collaboration
```

### Stream Interface

```go
// LEARN: Generic stream interface
// Works for any message type

type Stream[T any] interface {
    Send(msg T) error
    Recv() (T, error)
    Close() error
    Context() context.Context
}

// LEARN: Concrete implementation using channels
type ChannelStream[T any] struct {
    sendCh chan T
    recvCh chan T
    ctx    context.Context
    cancel context.CancelFunc
    closed int32
}

func NewChannelStream[T any](bufferSize int) *ChannelStream[T] {
    ctx, cancel := context.WithCancel(context.Background())
    return &ChannelStream[T]{
        sendCh: make(chan T, bufferSize),
        recvCh: make(chan T, bufferSize),
        ctx:    ctx,
        cancel: cancel,
    }
}
```

### Server-Side Streaming

```go
// LEARN: Server streaming pattern
// Server sends multiple responses to single request

type WatchResponse struct {
    Key       string
    Value     []byte
    Revision  int64
    EventType string  // "PUT" or "DELETE"
}

// Server-side handler
func (s *KVStore) Watch(req WatchRequest, stream Stream[WatchResponse]) error {
    // Create watcher for key prefix
    watcher := s.createWatcher(req.Prefix)
    defer watcher.Close()
    
    for {
        select {
        case <-stream.Context().Done():
            return stream.Context().Err()
            
        case event := <-watcher.Events():
            // LEARN: Send each event to client
            if err := stream.Send(WatchResponse{
                Key:       event.Key,
                Value:     event.Value,
                Revision:  event.Revision,
                EventType: event.Type,
            }); err != nil {
                return err
            }
        }
    }
}
```

### Client-Side Streaming

```go
// LEARN: Client streaming pattern
// Client sends multiple requests for single response

type BatchPutResponse struct {
    ItemsProcessed int
    Errors         []string
}

// Server-side handler
func (s *KVStore) BatchPut(stream Stream[PutRequest]) (*BatchPutResponse, error) {
    var processed int
    var errors []string
    
    // LEARN: Receive until client closes stream
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            break  // Client done sending
        }
        if err != nil {
            return nil, err
        }
        
        if err := s.put(req.Key, req.Value); err != nil {
            errors = append(errors, err.Error())
        } else {
            processed++
        }
    }
    
    return &BatchPutResponse{
        ItemsProcessed: processed,
        Errors:         errors,
    }, nil
}
```

### Bidirectional Streaming

```go
// LEARN: Bidirectional streaming
// Both sides send and receive concurrently

type ChatMessage struct {
    From    string
    Content string
    Time    time.Time
}

func (s *ChatServer) Chat(stream Stream[ChatMessage]) error {
    // LEARN: Handle send and receive in separate goroutines
    
    errCh := make(chan error, 2)
    
    // Receive goroutine
    go func() {
        for {
            msg, err := stream.Recv()
            if err != nil {
                errCh <- err
                return
            }
            
            // Broadcast to other clients
            s.broadcast(msg)
        }
    }()
    
    // Send goroutine - forward messages from other clients
    go func() {
        for msg := range s.getClientChannel(stream) {
            if err := stream.Send(msg); err != nil {
                errCh <- err
                return
            }
        }
    }()
    
    // Wait for either direction to error
    return <-errCh
}
```

### Flow Control

```go
// LEARN: Backpressure prevents overwhelming receivers
// Use buffered channels with select for non-blocking

type FlowControlledStream[T any] struct {
    *ChannelStream[T]
    maxPending int
    pending    int32
}

func (s *FlowControlledStream[T]) Send(msg T) error {
    // LEARN: Block if too many pending messages
    for atomic.LoadInt32(&s.pending) >= int32(s.maxPending) {
        select {
        case <-s.ctx.Done():
            return s.ctx.Err()
        case <-time.After(10 * time.Millisecond):
            // Retry
        }
    }
    
    atomic.AddInt32(&s.pending, 1)
    return s.ChannelStream.Send(msg)
}

func (s *FlowControlledStream[T]) Ack() {
    atomic.AddInt32(&s.pending, -1)
}
```

### Key Principles

- **Backpressure:** Slow consumers must not crash the system
- **Context Propagation:** Cancellation flows through the stream
- **Graceful Shutdown:** Close signals end of data
- **Error Handling:** Distinguish EOF from errors

---

## 3. Leader Election

**Files:** `pkg/distributed/leader.go`, `pkg/distributed/leader_test.go`

### Concept

Leader election selects a single coordinator among distributed nodes. This is simpler than full Raft when you only need coordination, not replicated state.

### What You'll Learn

```go
// LEARN: Bully algorithm - simplest leader election
// Higher ID wins, fast convergence

type LeaderElection struct {
    nodeID    string
    peers     []Peer
    leader    atomic.Value  // Current leader ID
    mu        sync.RWMutex
    isLeader  bool
}

func (le *LeaderElection) StartElection() {
    le.mu.Lock()
    defer le.mu.Unlock()
    
    // LEARN: Contact all nodes with higher IDs
    higherNodes := le.peersWithHigherID()
    
    if len(higherNodes) == 0 {
        // No higher nodes - become leader
        le.becomeLeader()
        return
    }
    
    // Send election messages to higher nodes
    for _, peer := range higherNodes {
        go func(p Peer) {
            if !p.Election(le.nodeID) {
                // Higher node responded - wait for coordinator message
                return
            }
        }(peer)
    }
}
```

### Lease-Based Election

```go
// LEARN: Lease-based leader election
// Leader holds lease, must renew periodically

type LeaderLease struct {
    nodeID     string
    leaseTime  time.Duration
    store      LeaseStore  // Distributed storage (etcd, consul, etc.)
    
    isLeader   bool
    renewCtx   context.Context
    renewStop  context.CancelFunc
}

// Attempt to acquire leadership
func (l *LeaderLease) Campaign(ctx context.Context) error {
    // LEARN: Try to acquire lease
    // Only one node can hold lease at a time
    
    acquired, err := l.store.TryAcquire(ctx, l.nodeID, l.leaseTime)
    if err != nil {
        return fmt.Errorf("lease acquire failed: %w", err)
    }
    
    if !acquired {
        return ErrNotLeader
    }
    
    l.isLeader = true
    l.renewCtx, l.renewStop = context.WithCancel(ctx)
    
    // LEARN: Start background renewal
    go l.renewLoop()
    
    return nil
}

func (l *LeaderLease) renewLoop() {
    ticker := time.NewTicker(l.leaseTime / 3)  // Renew at 1/3 of lease time
    defer ticker.Stop()
    
    for {
        select {
        case <-l.renewCtx.Done():
            return
            
        case <-ticker.C:
            if err := l.store.Renew(l.renewCtx, l.nodeID); err != nil {
                // LEARN: Lost leadership!
                l.isLeader = false
                l.onLostLeadership()
                return
            }
        }
    }
}
```

### Observing Leadership Changes

```go
// LEARN: Observer pattern for leadership changes
// Allows components to react to leader changes

type LeaderObserver interface {
    OnBecomeLeader()
    OnLoseLeadership()
    OnLeaderChanged(newLeader string)
}

type LeaderElectionWithObservers struct {
    *LeaderElection
    observers []LeaderObserver
    mu        sync.RWMutex
}

func (le *LeaderElectionWithObservers) AddObserver(o LeaderObserver) {
    le.mu.Lock()
    defer le.mu.Unlock()
    le.observers = append(le.observers, o)
}

func (le *LeaderElectionWithObservers) notifyBecomeLeader() {
    le.mu.RLock()
    defer le.mu.RUnlock()
    
    for _, o := range le.observers {
        // LEARN: Notify in goroutine to not block election
        go o.OnBecomeLeader()
    }
}
```

### Fencing Tokens

```go
// LEARN: Fencing prevents stale leaders from corrupting state
// Each leadership term has a unique token

type FencedLeader struct {
    *LeaderLease
    fenceToken uint64
}

func (f *FencedLeader) GetFenceToken() uint64 {
    return atomic.LoadUint64(&f.fenceToken)
}

// LEARN: Resource operations include fence token
// Server rejects operations with old tokens

type FencedRequest struct {
    FenceToken uint64
    Operation  string
    Data       []byte
}

func (s *Server) HandleFencedRequest(req FencedRequest) error {
    // LEARN: Reject stale requests
    if req.FenceToken < s.lastKnownToken {
        return ErrStaleFenceToken
    }
    
    s.lastKnownToken = req.FenceToken
    return s.process(req)
}
```

### Key Principles

- **Single Leader:** Only one active leader at any time
- **Lease Renewal:** Leadership must be actively maintained
- **Fencing:** Prevent split-brain with monotonic tokens
- **Observer Pattern:** React to leadership changes

---

## 4. Log Replication

**Files:** `pkg/distributed/log.go`, `pkg/distributed/log_test.go`

### Concept

Log replication ensures all nodes have the same sequence of operations. This is the foundation for replicated state machines.

### What You'll Learn

```go
// LEARN: Append-only log structure
// Core data structure for replication

type ReplicatedLog struct {
    entries []LogEntry
    mu      sync.RWMutex
    
    // LEARN: Commit index is the highest entry known to be
    // replicated to a majority of nodes
    commitIndex uint64
    
    // LEARN: Applied index is what's been applied to state machine
    // Always: appliedIndex <= commitIndex <= lastIndex
    appliedIndex uint64
}

type LogEntry struct {
    Index   uint64
    Term    uint64
    Command []byte
}
```

### Log Operations

```go
// LEARN: Safe append with consistency check
func (l *ReplicatedLog) Append(entries []LogEntry) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    for _, entry := range entries {
        // LEARN: Check for gaps
        if entry.Index != uint64(len(l.entries)) {
            return fmt.Errorf("log gap: expected index %d, got %d",
                len(l.entries), entry.Index)
        }
        l.entries = append(l.entries, entry)
    }
    
    return nil
}

// LEARN: Truncate removes inconsistent entries
// Called when follower's log diverges from leader's
func (l *ReplicatedLog) Truncate(fromIndex uint64) error {
    l.mu.Lock()
    defer l.mu.Unlock()
    
    if fromIndex >= uint64(len(l.entries)) {
        return nil  // Nothing to truncate
    }
    
    // LEARN: Never truncate committed entries!
    if fromIndex <= l.commitIndex {
        return ErrCannotTruncateCommitted
    }
    
    l.entries = l.entries[:fromIndex]
    return nil
}
```

### Persistence

```go
// LEARN: WAL (Write-Ahead Log) for durability
// Entries are written to disk before acknowledgment

type WAL struct {
    file     *os.File
    encoder  *gob.Encoder
    mu       sync.Mutex
    syncMode SyncMode
}

type SyncMode int

const (
    SyncNone  SyncMode = iota  // Fastest, may lose data
    SyncBatch                   // Sync periodically
    SyncEvery                   // Sync each write (safest)
)

func (w *WAL) Write(entry LogEntry) error {
    w.mu.Lock()
    defer w.mu.Unlock()
    
    // LEARN: Encode entry to disk
    if err := w.encoder.Encode(entry); err != nil {
        return fmt.Errorf("encode failed: %w", err)
    }
    
    // LEARN: Sync based on durability requirements
    if w.syncMode == SyncEvery {
        if err := w.file.Sync(); err != nil {
            return fmt.Errorf("sync failed: %w", err)
        }
    }
    
    return nil
}
```

### Snapshotting

```go
// LEARN: Snapshots prevent unbounded log growth
// Capture state machine state, discard old log entries

type Snapshot struct {
    LastIncludedIndex uint64
    LastIncludedTerm  uint64
    Data              []byte
}

type SnapshotManager struct {
    log           *ReplicatedLog
    stateMachine  StateMachine
    snapshotDir   string
    mu            sync.Mutex
}

func (sm *SnapshotManager) CreateSnapshot() (*Snapshot, error) {
    sm.mu.Lock()
    defer sm.mu.Unlock()
    
    // LEARN: Snapshot state machine at current applied index
    appliedIndex := sm.log.AppliedIndex()
    term := sm.log.Term(appliedIndex)
    
    data, err := sm.stateMachine.Serialize()
    if err != nil {
        return nil, fmt.Errorf("serialize state: %w", err)
    }
    
    snapshot := &Snapshot{
        LastIncludedIndex: appliedIndex,
        LastIncludedTerm:  term,
        Data:              data,
    }
    
    // LEARN: Persist snapshot, then compact log
    if err := sm.persist(snapshot); err != nil {
        return nil, err
    }
    
    sm.log.DiscardBefore(appliedIndex + 1)
    
    return snapshot, nil
}
```

### Replication Protocol

```go
// LEARN: Leader replicates to followers
// Track progress per follower

type ReplicationManager struct {
    log      *ReplicatedLog
    peers    map[string]*PeerState
    commitCh chan uint64
}

type PeerState struct {
    nextIndex  uint64  // Next entry to send to this peer
    matchIndex uint64  // Highest entry known to be replicated
}

func (rm *ReplicationManager) replicate(peerID string, peer Peer) {
    state := rm.peers[peerID]
    
    for {
        // LEARN: Get entries to send
        entries := rm.log.EntriesFrom(state.nextIndex)
        
        if len(entries) == 0 {
            time.Sleep(10 * time.Millisecond)
            continue
        }
        
        // Send AppendEntries RPC
        reply, err := peer.AppendEntries(AppendEntriesRequest{
            PrevLogIndex: state.nextIndex - 1,
            PrevLogTerm:  rm.log.Term(state.nextIndex - 1),
            Entries:      entries,
            LeaderCommit: rm.log.CommitIndex(),
        })
        
        if err != nil {
            continue  // Retry
        }
        
        if reply.Success {
            // LEARN: Update progress
            state.matchIndex = entries[len(entries)-1].Index
            state.nextIndex = state.matchIndex + 1
            
            rm.updateCommitIndex()
        } else {
            // LEARN: Decrement and retry (log mismatch)
            state.nextIndex--
        }
    }
}

func (rm *ReplicationManager) updateCommitIndex() {
    // LEARN: Commit index is highest entry replicated to majority
    matchIndexes := make([]uint64, 0, len(rm.peers)+1)
    matchIndexes = append(matchIndexes, rm.log.LastIndex())  // Leader
    
    for _, state := range rm.peers {
        matchIndexes = append(matchIndexes, state.matchIndex)
    }
    
    sort.Slice(matchIndexes, func(i, j int) bool {
        return matchIndexes[i] > matchIndexes[j]
    })
    
    // Majority index
    majorityIdx := len(matchIndexes) / 2
    newCommit := matchIndexes[majorityIdx]
    
    if newCommit > rm.log.CommitIndex() {
        rm.log.SetCommitIndex(newCommit)
        rm.commitCh <- newCommit
    }
}
```

### Key Principles

- **Append-Only:** Entries are never modified, only appended
- **Consistency Check:** Verify previous entry before appending
- **Durability:** Write to disk before acknowledging
- **Compaction:** Snapshots prevent unbounded growth

---

## 5. Building Distributed Systems

**Files:** `pkg/distributed/kv.go`, `pkg/distributed/kv_test.go`

### Concept

Combine all components into a working distributed key-value store.

### What You'll Learn

```go
// LEARN: Distributed KV store architecture
// Combines: Raft + Log Replication + State Machine

type DistributedKV struct {
    raft      *RaftNode
    log       *ReplicatedLog
    state     *KVStateMachine
    proposals chan Proposal
}

type KVStateMachine struct {
    data map[string][]byte
    mu   sync.RWMutex
}

// LEARN: Apply committed log entries to state machine
func (kv *KVStateMachine) Apply(entry LogEntry) error {
    kv.mu.Lock()
    defer kv.mu.Unlock()
    
    var cmd KVCommand
    if err := json.Unmarshal(entry.Command, &cmd); err != nil {
        return err
    }
    
    switch cmd.Op {
    case "PUT":
        kv.data[cmd.Key] = cmd.Value
    case "DELETE":
        delete(kv.data, cmd.Key)
    }
    
    return nil
}
```

### Client Interface

```go
// LEARN: Linearizable reads and writes
// All operations go through leader

func (d *DistributedKV) Put(ctx context.Context, key string, value []byte) error {
    // LEARN: Only leader can accept writes
    if !d.raft.IsLeader() {
        return ErrNotLeader
    }
    
    cmd := KVCommand{Op: "PUT", Key: key, Value: value}
    data, _ := json.Marshal(cmd)
    
    // LEARN: Propose to Raft, wait for commit
    return d.raft.Propose(ctx, data)
}

func (d *DistributedKV) Get(ctx context.Context, key string) ([]byte, error) {
    // LEARN: For linearizable read, verify leadership
    if !d.raft.IsLeader() {
        // Option 1: Redirect to leader
        // Option 2: Read from follower (stale read)
        return nil, ErrNotLeader
    }
    
    // LEARN: ReadIndex ensures we're still leader
    if err := d.raft.ReadIndex(ctx); err != nil {
        return nil, err
    }
    
    return d.state.Get(key), nil
}
```

### Key Principles

- **Replicated State Machine:** Same commands → same state
- **Linearizability:** Operations appear atomic and instantaneous
- **Leadership:** All writes through single leader
- **Read Options:** Strong vs eventual consistency tradeoff

---

## Summary: Phase 7 Competencies

After completing Phase 7, you can:

| Skill | Assessment |
|-------|------------|
| Implement Raft basics | Election, heartbeat, log replication |
| Build streaming APIs | Server, client, bidirectional streams |
| Design leader election | Lease-based, fencing tokens |
| Create replicated logs | Append, truncate, snapshot |
| Build distributed KV | Combine all components |

---

## Exercises Checklist

- [ ] **Exercise 7.1:** Simplified Raft
  - Implements: Leader election, heartbeats, basic log replication
  - File: `pkg/distributed/raft.go`
  - Test: `pkg/distributed/raft_test.go`
  - Requirements:
    - Node state transitions (Follower → Candidate → Leader)
    - Vote request/grant with term checking
    - Heartbeat mechanism
    - Basic log entry append

- [ ] **Exercise 7.2:** Streaming Service
  - Implements: Generic streaming primitives
  - File: `pkg/distributed/stream.go`
  - Test: `pkg/distributed/stream_test.go`
  - Requirements:
    - Server-side streaming (one-to-many)
    - Client-side streaming (many-to-one)
    - Bidirectional streaming
    - Flow control with backpressure

- [ ] **Exercise 7.3:** Leader Election Service
  - Implements: Lease-based leader election
  - File: `pkg/distributed/leader.go`
  - Test: `pkg/distributed/leader_test.go`
  - Requirements:
    - Lease acquisition and renewal
    - Leadership observation
    - Fencing tokens
    - Graceful leadership transfer

---

## Key Papers and Resources

1. **Raft Paper:** "In Search of an Understandable Consensus Algorithm" (Ongaro & Ousterhout)
2. **FLP Impossibility:** Understanding why consensus is hard
3. **Paxos Made Simple:** Lamport's simplified Paxos explanation
4. **etcd/raft:** Production Raft implementation in Go

---

## Next Steps

Congratulations! You've completed all seven phases of GoLlama Runtime. You now have:
- Deep Go language knowledge (Phases 1-3)
- Performance optimization skills (Phase 4)
- Systems programming expertise (Phase 5)
- Tooling development ability (Phase 6)
- Distributed systems understanding (Phase 7)

These skills prepare you for building production distributed systems, contributing to projects like etcd, CockroachDB, or Kubernetes.


