// Package distributed provides distributed systems primitives including
// consensus algorithms, streaming, and leader election.
//
// LEARN: This package teaches distributed systems concepts from scratch.
// Phase 7 builds on all previous phases to create systems that work
// across multiple nodes with fault tolerance.
package distributed

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// LEARN: Raft node states
// Every node is in exactly one of these states at any time.
// The state machine is: Follower <-> Candidate <-> Leader

// NodeState represents the current state of a Raft node.
type NodeState int32

const (
	// Follower is the default state. Followers accept log entries from the leader.
	Follower NodeState = iota
	// Candidate is a transient state during leader election.
	Candidate
	// Leader handles all client requests and replicates to followers.
	Leader
)

func (s NodeState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		return "Unknown"
	}
}

// LEARN: Configuration for Raft timing
// These values affect election stability and replication latency.

// RaftConfig contains configuration for a Raft node.
type RaftConfig struct {
	// NodeID uniquely identifies this node in the cluster.
	NodeID string

	// ElectionTimeout is the time a follower waits before starting election.
	// Randomized between ElectionTimeout and ElectionTimeout*2 for stability.
	ElectionTimeout time.Duration

	// HeartbeatInterval is how often the leader sends heartbeats.
	// Must be much smaller than ElectionTimeout.
	HeartbeatInterval time.Duration

	// Peers is the list of other nodes in the cluster.
	Peers []string
}

// DefaultRaftConfig returns sensible defaults for Raft configuration.
func DefaultRaftConfig(nodeID string, peers []string) RaftConfig {
	return RaftConfig{
		NodeID:            nodeID,
		ElectionTimeout:   150 * time.Millisecond,
		HeartbeatInterval: 50 * time.Millisecond,
		Peers:             peers,
	}
}

// LEARN: Log entries are the heart of Raft
// Each entry contains a command to be applied to the state machine.

// LogEntry represents a single entry in the Raft log.
type LogEntry struct {
	// Index is the position in the log (1-indexed).
	Index uint64

	// Term is the leader's term when entry was created.
	// LEARN: Terms are logical clocks that help detect stale leaders.
	Term uint64

	// Command is the operation to apply to the state machine.
	Command []byte
}

// LEARN: RPC messages for Raft protocol
// These are the only two message types needed for basic Raft.

// RequestVoteArgs contains arguments for RequestVote RPC.
type RequestVoteArgs struct {
	// Term is the candidate's term.
	Term uint64

	// CandidateID is the candidate requesting vote.
	CandidateID string

	// LastLogIndex is index of candidate's last log entry.
	LastLogIndex uint64

	// LastLogTerm is term of candidate's last log entry.
	LastLogTerm uint64
}

// RequestVoteReply contains the reply to RequestVote RPC.
type RequestVoteReply struct {
	// Term is the current term for candidate to update itself.
	Term uint64

	// VoteGranted is true if candidate received vote.
	VoteGranted bool
}

// AppendEntriesArgs contains arguments for AppendEntries RPC.
type AppendEntriesArgs struct {
	// Term is leader's term.
	Term uint64

	// LeaderID so follower can redirect clients.
	LeaderID string

	// PrevLogIndex is index of log entry immediately preceding new ones.
	PrevLogIndex uint64

	// PrevLogTerm is term of prevLogIndex entry.
	PrevLogTerm uint64

	// Entries are log entries to store (empty for heartbeat).
	Entries []LogEntry

	// LeaderCommit is leader's commitIndex.
	LeaderCommit uint64
}

// AppendEntriesReply contains the reply to AppendEntries RPC.
type AppendEntriesReply struct {
	// Term is current term for leader to update itself.
	Term uint64

	// Success is true if follower contained entry matching prevLogIndex and prevLogTerm.
	Success bool

	// ConflictIndex is the first index where log differs (for fast backup).
	ConflictIndex uint64

	// ConflictTerm is the term of the conflicting entry.
	ConflictTerm uint64
}

// LEARN: Transport interface abstracts network communication
// This allows testing with in-memory transport and production with real network.

// RaftTransport handles communication between Raft nodes.
type RaftTransport interface {
	// RequestVote sends a vote request to a peer.
	RequestVote(ctx context.Context, peer string, args RequestVoteArgs) (RequestVoteReply, error)

	// AppendEntries sends log entries (or heartbeat) to a peer.
	AppendEntries(ctx context.Context, peer string, args AppendEntriesArgs) (AppendEntriesReply, error)
}

// LEARN: State machine interface for applying committed entries
// Raft is just consensus - your application provides the state machine.

// StateMachine is applied with committed log entries.
type StateMachine interface {
	// Apply applies a committed log entry to the state machine.
	Apply(entry LogEntry) error
}

// RaftNode implements a simplified Raft consensus node.
type RaftNode struct {
	// LEARN: Persistent state (must survive restarts)
	// In production, these would be written to disk.
	currentTerm uint64
	votedFor    string
	log         []LogEntry

	// LEARN: Volatile state (can be reconstructed)
	commitIndex uint64 // Highest log entry known to be committed
	lastApplied uint64 // Highest log entry applied to state machine

	// LEARN: Leader-only state (reinitialized after election)
	nextIndex  map[string]uint64 // For each peer: next log index to send
	matchIndex map[string]uint64 // For each peer: highest log index known to be replicated

	// Node state
	state    int32 // NodeState as atomic
	leaderID string

	// Configuration
	config    RaftConfig
	transport RaftTransport
	sm        StateMachine

	// Channels
	applyCh     chan LogEntry
	commitCh    chan struct{}
	stepDownCh  chan struct{}
	heartbeatCh chan struct{}
	stopCh      chan struct{}

	// Synchronization
	mu sync.RWMutex

	// Metrics
	electionCount uint64
}

// NewRaftNode creates a new Raft node with the given configuration.
func NewRaftNode(config RaftConfig, transport RaftTransport, sm StateMachine) *RaftNode {
	n := &RaftNode{
		currentTerm: 0,
		votedFor:    "",
		log:         make([]LogEntry, 0),
		commitIndex: 0,
		lastApplied: 0,
		nextIndex:   make(map[string]uint64),
		matchIndex:  make(map[string]uint64),
		state:       int32(Follower),
		config:      config,
		transport:   transport,
		sm:          sm,
		applyCh:     make(chan LogEntry, 100),
		commitCh:    make(chan struct{}, 1),
		stepDownCh:  make(chan struct{}, 1),
		heartbeatCh: make(chan struct{}, 1),
		stopCh:      make(chan struct{}),
	}

	// LEARN: Initialize log with a dummy entry at index 0
	// This simplifies log indexing (real entries start at 1)
	n.log = append(n.log, LogEntry{Index: 0, Term: 0})

	return n
}

// Start begins the Raft node's main loop.
func (n *RaftNode) Start() {
	go n.run()
	go n.applyLoop()
}

// Stop gracefully shuts down the Raft node.
func (n *RaftNode) Stop() {
	close(n.stopCh)
}

// LEARN: Main event loop
// Different behavior based on current state.

func (n *RaftNode) run() {
	for {
		select {
		case <-n.stopCh:
			return
		default:
		}

		switch n.getState() {
		case Follower:
			n.runFollower()
		case Candidate:
			n.runCandidate()
		case Leader:
			n.runLeader()
		}
	}
}

// LEARN: Follower behavior
// Wait for heartbeats, start election on timeout.

func (n *RaftNode) runFollower() {
	timeout := n.randomElectionTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	for n.getState() == Follower {
		select {
		case <-n.stopCh:
			return

		case <-n.heartbeatCh:
			// LEARN: Reset timer on valid heartbeat
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(n.randomElectionTimeout())

		case <-timer.C:
			// LEARN: No heartbeat - start election
			n.setState(Candidate)
			return

		case <-n.stepDownCh:
			// Another node has higher term
			return
		}
	}
}

// LEARN: Candidate behavior
// Request votes, become leader or step down.

func (n *RaftNode) runCandidate() {
	// LEARN: Increment term and vote for self
	n.mu.Lock()
	n.currentTerm++
	n.votedFor = n.config.NodeID
	currentTerm := n.currentTerm
	lastLogIndex, lastLogTerm := n.lastLogInfo()
	n.mu.Unlock()

	atomic.AddUint64(&n.electionCount, 1)

	// Start election timer
	timeout := n.randomElectionTimeout()
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	// LEARN: Request votes in parallel
	votes := int32(1) // Already have own vote
	voteCh := make(chan bool, len(n.config.Peers))

	for _, peer := range n.config.Peers {
		go func(p string) {
			reply, err := n.transport.RequestVote(
				context.Background(),
				p,
				RequestVoteArgs{
					Term:         currentTerm,
					CandidateID:  n.config.NodeID,
					LastLogIndex: lastLogIndex,
					LastLogTerm:  lastLogTerm,
				},
			)

			if err != nil {
				voteCh <- false
				return
			}

			// LEARN: Step down if we see higher term
			if reply.Term > currentTerm {
				n.mu.Lock()
				if reply.Term > n.currentTerm {
					n.currentTerm = reply.Term
					n.votedFor = ""
				}
				n.mu.Unlock()
				n.setState(Follower)
				voteCh <- false
				return
			}

			voteCh <- reply.VoteGranted
		}(peer)
	}

	// Collect votes
	for n.getState() == Candidate {
		select {
		case <-n.stopCh:
			return

		case <-timer.C:
			// LEARN: Election timeout - start new election
			return

		case <-n.stepDownCh:
			n.setState(Follower)
			return

		case granted := <-voteCh:
			if granted {
				newVotes := atomic.AddInt32(&votes, 1)
				// LEARN: Majority = become leader
				if int(newVotes) > (len(n.config.Peers)+1)/2 {
					n.becomeLeader()
					return
				}
			}
		}
	}
}

// LEARN: Leader behavior
// Send heartbeats, replicate log entries.

func (n *RaftNode) runLeader() {
	// Initialize leader state
	n.mu.Lock()
	n.leaderID = n.config.NodeID
	lastIndex := n.lastLogIndex()
	for _, peer := range n.config.Peers {
		n.nextIndex[peer] = lastIndex + 1
		n.matchIndex[peer] = 0
	}
	n.mu.Unlock()

	// Send initial heartbeats
	n.broadcastHeartbeat()

	ticker := time.NewTicker(n.config.HeartbeatInterval)
	defer ticker.Stop()

	for n.getState() == Leader {
		select {
		case <-n.stopCh:
			return

		case <-ticker.C:
			n.broadcastHeartbeat()

		case <-n.stepDownCh:
			n.setState(Follower)
			return

		case <-n.commitCh:
			n.updateCommitIndex()
		}
	}
}

func (n *RaftNode) broadcastHeartbeat() {
	n.mu.RLock()
	term := n.currentTerm
	leaderID := n.config.NodeID
	commitIndex := n.commitIndex
	peers := n.config.Peers
	n.mu.RUnlock()

	for _, peer := range peers {
		go func(p string) {
			n.mu.RLock()
			nextIdx := n.nextIndex[p]
			prevLogIndex := nextIdx - 1
			prevLogTerm := uint64(0)
			if prevLogIndex < uint64(len(n.log)) {
				prevLogTerm = n.log[prevLogIndex].Term
			}

			// Get entries to send
			var entries []LogEntry
			if nextIdx <= uint64(len(n.log))-1 {
				entries = make([]LogEntry, len(n.log)-int(nextIdx))
				copy(entries, n.log[nextIdx:])
			}
			n.mu.RUnlock()

			reply, err := n.transport.AppendEntries(
				context.Background(),
				p,
				AppendEntriesArgs{
					Term:         term,
					LeaderID:     leaderID,
					PrevLogIndex: prevLogIndex,
					PrevLogTerm:  prevLogTerm,
					Entries:      entries,
					LeaderCommit: commitIndex,
				},
			)

			if err != nil {
				return
			}

			n.mu.Lock()
			defer n.mu.Unlock()

			// LEARN: Step down if higher term discovered
			if reply.Term > n.currentTerm {
				n.currentTerm = reply.Term
				n.votedFor = ""
				n.setState(Follower)
				return
			}

			if reply.Success {
				if len(entries) > 0 {
					n.matchIndex[p] = entries[len(entries)-1].Index
					n.nextIndex[p] = n.matchIndex[p] + 1
				}
				// Signal to update commit index
				select {
				case n.commitCh <- struct{}{}:
				default:
				}
			} else {
				// LEARN: Decrement nextIndex and retry
				// Fast backup using conflict info
				if reply.ConflictIndex > 0 {
					n.nextIndex[p] = reply.ConflictIndex
				} else if n.nextIndex[p] > 1 {
					n.nextIndex[p]--
				}
			}
		}(peer)
	}
}

// LEARN: Update commit index when majority has replicated
func (n *RaftNode) updateCommitIndex() {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Collect all match indexes including self
	matchIndexes := make([]uint64, 0, len(n.config.Peers)+1)
	matchIndexes = append(matchIndexes, n.lastLogIndex()) // Leader

	for _, peer := range n.config.Peers {
		matchIndexes = append(matchIndexes, n.matchIndex[peer])
	}

	// Sort descending
	for i := 0; i < len(matchIndexes)-1; i++ {
		for j := i + 1; j < len(matchIndexes); j++ {
			if matchIndexes[j] > matchIndexes[i] {
				matchIndexes[i], matchIndexes[j] = matchIndexes[j], matchIndexes[i]
			}
		}
	}

	// LEARN: Majority index (N/2 + 1 nodes have this or higher)
	majorityIdx := len(matchIndexes) / 2
	newCommit := matchIndexes[majorityIdx]

	// LEARN: Only commit entries from current term
	// This prevents committing entries from old terms until
	// we've committed something from current term
	if newCommit > n.commitIndex && newCommit < uint64(len(n.log)) {
		if n.log[newCommit].Term == n.currentTerm {
			n.commitIndex = newCommit
		}
	}
}

// applyLoop applies committed entries to the state machine.
func (n *RaftNode) applyLoop() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-n.stopCh:
			return
		case <-ticker.C:
			n.applyCommitted()
		}
	}
}

func (n *RaftNode) applyCommitted() {
	n.mu.Lock()
	commitIndex := n.commitIndex
	lastApplied := n.lastApplied

	var toApply []LogEntry
	for i := lastApplied + 1; i <= commitIndex && i < uint64(len(n.log)); i++ {
		toApply = append(toApply, n.log[i])
	}
	n.mu.Unlock()

	for _, entry := range toApply {
		if n.sm != nil {
			if err := n.sm.Apply(entry); err != nil {
				// Log error but continue - committed entries must be applied
				continue
			}
		}

		n.mu.Lock()
		n.lastApplied = entry.Index
		n.mu.Unlock()

		// Notify via channel
		select {
		case n.applyCh <- entry:
		default:
		}
	}
}

// HandleRequestVote processes an incoming RequestVote RPC.
func (n *RaftNode) HandleRequestVote(args RequestVoteArgs) RequestVoteReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := RequestVoteReply{
		Term:        n.currentTerm,
		VoteGranted: false,
	}

	// LEARN: Reject if candidate's term is older
	if args.Term < n.currentTerm {
		return reply
	}

	// LEARN: Step down if newer term
	if args.Term > n.currentTerm {
		n.currentTerm = args.Term
		n.votedFor = ""
		n.setState(Follower)
	}

	reply.Term = n.currentTerm

	// LEARN: Grant vote if:
	// 1. Haven't voted yet OR already voted for this candidate
	// 2. Candidate's log is at least as up-to-date as ours
	if (n.votedFor == "" || n.votedFor == args.CandidateID) &&
		n.isLogUpToDate(args.LastLogIndex, args.LastLogTerm) {
		n.votedFor = args.CandidateID
		reply.VoteGranted = true

		// Reset election timer
		select {
		case n.heartbeatCh <- struct{}{}:
		default:
		}
	}

	return reply
}

// LEARN: Log comparison for voting
// Candidate must have log at least as up-to-date as voter.
func (n *RaftNode) isLogUpToDate(lastIndex, lastTerm uint64) bool {
	myLastIndex, myLastTerm := n.lastLogInfo()

	// LEARN: Compare by term first, then by index
	if lastTerm != myLastTerm {
		return lastTerm >= myLastTerm
	}
	return lastIndex >= myLastIndex
}

// HandleAppendEntries processes an incoming AppendEntries RPC.
func (n *RaftNode) HandleAppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	n.mu.Lock()
	defer n.mu.Unlock()

	reply := AppendEntriesReply{
		Term:    n.currentTerm,
		Success: false,
	}

	// LEARN: Reject if leader's term is older
	if args.Term < n.currentTerm {
		return reply
	}

	// LEARN: Valid leader - reset election timer
	select {
	case n.heartbeatCh <- struct{}{}:
	default:
	}

	// Step down if newer or equal term
	if args.Term >= n.currentTerm {
		if args.Term > n.currentTerm {
			n.currentTerm = args.Term
			n.votedFor = ""
		}
		n.leaderID = args.LeaderID
		if n.getState() != Follower {
			n.setState(Follower)
		}
	}

	reply.Term = n.currentTerm

	// LEARN: Log consistency check
	// Verify we have entry at prevLogIndex with matching term
	if args.PrevLogIndex > 0 {
		if args.PrevLogIndex >= uint64(len(n.log)) {
			// We don't have this entry yet
			reply.ConflictIndex = uint64(len(n.log))
			return reply
		}
		if n.log[args.PrevLogIndex].Term != args.PrevLogTerm {
			// Term mismatch - find first entry with conflicting term
			reply.ConflictTerm = n.log[args.PrevLogIndex].Term
			for i := args.PrevLogIndex; i > 0; i-- {
				if n.log[i-1].Term != reply.ConflictTerm {
					reply.ConflictIndex = i
					break
				}
			}
			if reply.ConflictIndex == 0 {
				reply.ConflictIndex = 1
			}
			return reply
		}
	}

	// LEARN: Append new entries
	for i, entry := range args.Entries {
		idx := args.PrevLogIndex + uint64(i) + 1

		if idx < uint64(len(n.log)) {
			if n.log[idx].Term != entry.Term {
				// LEARN: Conflict - truncate and append
				n.log = n.log[:idx]
				n.log = append(n.log, entry)
			}
			// Same entry - skip
		} else {
			// New entry
			n.log = append(n.log, entry)
		}
	}

	// LEARN: Update commit index
	if args.LeaderCommit > n.commitIndex {
		lastNewIndex := args.PrevLogIndex + uint64(len(args.Entries))
		if args.LeaderCommit < lastNewIndex {
			n.commitIndex = args.LeaderCommit
		} else {
			n.commitIndex = lastNewIndex
		}
	}

	reply.Success = true
	return reply
}

// Propose submits a command to be replicated (leader only).
func (n *RaftNode) Propose(ctx context.Context, command []byte) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.getState() != Leader {
		return ErrNotLeader
	}

	// Create new log entry
	entry := LogEntry{
		Index:   uint64(len(n.log)),
		Term:    n.currentTerm,
		Command: command,
	}

	n.log = append(n.log, entry)

	return nil
}

// Helper methods

func (n *RaftNode) becomeLeader() {
	n.setState(Leader)
}

func (n *RaftNode) getState() NodeState {
	return NodeState(atomic.LoadInt32(&n.state))
}

func (n *RaftNode) setState(s NodeState) {
	atomic.StoreInt32(&n.state, int32(s))
}

func (n *RaftNode) lastLogIndex() uint64 {
	return uint64(len(n.log)) - 1
}

func (n *RaftNode) lastLogInfo() (index, term uint64) {
	if len(n.log) == 0 {
		return 0, 0
	}
	last := n.log[len(n.log)-1]
	return last.Index, last.Term
}

func (n *RaftNode) randomElectionTimeout() time.Duration {
	// LEARN: Randomization prevents split votes
	base := n.config.ElectionTimeout
	jitter := time.Duration(rand.Int63n(int64(base)))
	return base + jitter
}

// Public getters for node state

// State returns the current state of the node.
func (n *RaftNode) State() NodeState {
	return n.getState()
}

// IsLeader returns true if this node is the current leader.
func (n *RaftNode) IsLeader() bool {
	return n.getState() == Leader
}

// LeaderID returns the current leader's ID.
func (n *RaftNode) LeaderID() string {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.leaderID
}

// CurrentTerm returns the current term.
func (n *RaftNode) CurrentTerm() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.currentTerm
}

// CommitIndex returns the current commit index.
func (n *RaftNode) CommitIndex() uint64 {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.commitIndex
}

// ApplyCh returns a channel that receives applied log entries.
func (n *RaftNode) ApplyCh() <-chan LogEntry {
	return n.applyCh
}

// ElectionCount returns the number of elections started by this node.
func (n *RaftNode) ElectionCount() uint64 {
	return atomic.LoadUint64(&n.electionCount)
}

// Errors for Raft operations
var (
	ErrNotLeader = fmt.Errorf("not the leader")
)

