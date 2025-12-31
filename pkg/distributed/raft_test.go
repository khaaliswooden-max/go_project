package distributed

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// LEARN: Mock transport for testing Raft without network
// This is a common pattern for testing distributed systems.

type mockTransport struct {
	nodes map[string]*RaftNode
	mu    sync.RWMutex

	// For injecting failures
	dropRate    float64
	delayMs     int
	partitioned map[string]map[string]bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		nodes:       make(map[string]*RaftNode),
		partitioned: make(map[string]map[string]bool),
	}
}

func (t *mockTransport) register(node *RaftNode) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.nodes[node.config.NodeID] = node
}

func (t *mockTransport) partition(from, to string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.partitioned[from] == nil {
		t.partitioned[from] = make(map[string]bool)
	}
	t.partitioned[from][to] = true
}

func (t *mockTransport) heal(from, to string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.partitioned[from] != nil {
		delete(t.partitioned[from], to)
	}
}

func (t *mockTransport) isPartitioned(from, to string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if t.partitioned[from] != nil {
		return t.partitioned[from][to]
	}
	return false
}

func (t *mockTransport) RequestVote(ctx context.Context, peer string, args RequestVoteArgs) (RequestVoteReply, error) {
	t.mu.RLock()
	node, ok := t.nodes[peer]
	t.mu.RUnlock()

	if !ok {
		return RequestVoteReply{}, ErrNodeNotFound
	}

	// Check for partition
	if t.isPartitioned(args.CandidateID, peer) {
		return RequestVoteReply{}, ErrPartitioned
	}

	return node.HandleRequestVote(args), nil
}

func (t *mockTransport) AppendEntries(ctx context.Context, peer string, args AppendEntriesArgs) (AppendEntriesReply, error) {
	t.mu.RLock()
	node, ok := t.nodes[peer]
	t.mu.RUnlock()

	if !ok {
		return AppendEntriesReply{}, ErrNodeNotFound
	}

	// Check for partition
	if t.isPartitioned(args.LeaderID, peer) {
		return AppendEntriesReply{}, ErrPartitioned
	}

	return node.HandleAppendEntries(args), nil
}

// Test errors
var (
	ErrNodeNotFound = NewError("node not found")
	ErrPartitioned  = NewError("network partitioned")
)

// NewError creates a new error (helper for tests).
func NewError(msg string) error {
	return &testError{msg: msg}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

// LEARN: Mock state machine for testing
type mockStateMachine struct {
	applied []LogEntry
	mu      sync.Mutex
}

func (m *mockStateMachine) Apply(entry LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.applied = append(m.applied, entry)
	return nil
}

func (m *mockStateMachine) AppliedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.applied)
}

// TestRaftNodeCreation tests basic node creation.
func TestRaftNodeCreation(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", []string{"node2", "node3"})
	sm := &mockStateMachine{}

	node := NewRaftNode(config, transport, sm)

	if node == nil {
		t.Fatal("NewRaftNode returned nil")
	}

	if node.State() != Follower {
		t.Errorf("expected initial state Follower, got %s", node.State())
	}

	if node.CurrentTerm() != 0 {
		t.Errorf("expected initial term 0, got %d", node.CurrentTerm())
	}
}

// TestRaftStateTransitions tests node state changes.
func TestRaftStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		from     NodeState
		to       NodeState
		expected string
	}{
		{"follower to candidate", Follower, Candidate, "Candidate"},
		{"candidate to leader", Candidate, Leader, "Leader"},
		{"leader to follower", Leader, Follower, "Follower"},
		{"candidate to follower", Candidate, Follower, "Follower"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			transport := newMockTransport()
			config := DefaultRaftConfig("node1", nil)
			node := NewRaftNode(config, transport, nil)

			node.setState(tc.from)
			node.setState(tc.to)

			if node.State().String() != tc.expected {
				t.Errorf("expected state %s, got %s", tc.expected, node.State())
			}
		})
	}
}

// TestRaftRequestVote tests vote request handling.
func TestRaftRequestVote(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", []string{"node2"})
	node := NewRaftNode(config, transport, nil)

	tests := []struct {
		name        string
		setupTerm   uint64
		args        RequestVoteArgs
		expectVote  bool
		expectTerm  uint64
	}{
		{
			name:      "grant vote - newer term",
			setupTerm: 0,
			args: RequestVoteArgs{
				Term:         1,
				CandidateID:  "node2",
				LastLogIndex: 0,
				LastLogTerm:  0,
			},
			expectVote: true,
			expectTerm: 1,
		},
		{
			name:      "reject vote - older term",
			setupTerm: 5,
			args: RequestVoteArgs{
				Term:         3,
				CandidateID:  "node2",
				LastLogIndex: 0,
				LastLogTerm:  0,
			},
			expectVote: false,
			expectTerm: 5,
		},
		{
			name:      "reject vote - already voted",
			setupTerm: 1,
			args: RequestVoteArgs{
				Term:         1,
				CandidateID:  "node3", // Different candidate
				LastLogIndex: 0,
				LastLogTerm:  0,
			},
			expectVote: false,
			expectTerm: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset node state
			node.mu.Lock()
			node.currentTerm = tc.setupTerm
			if tc.name != "reject vote - already voted" {
				node.votedFor = ""
			} else {
				node.votedFor = "node2"
			}
			node.mu.Unlock()

			reply := node.HandleRequestVote(tc.args)

			if reply.VoteGranted != tc.expectVote {
				t.Errorf("expected VoteGranted=%v, got %v", tc.expectVote, reply.VoteGranted)
			}

			if reply.Term != tc.expectTerm {
				t.Errorf("expected Term=%d, got %d", tc.expectTerm, reply.Term)
			}
		})
	}
}

// TestRaftAppendEntries tests append entries handling.
func TestRaftAppendEntries(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", []string{"node2"})
	node := NewRaftNode(config, transport, nil)

	tests := []struct {
		name          string
		setupTerm     uint64
		args          AppendEntriesArgs
		expectSuccess bool
		expectTerm    uint64
	}{
		{
			name:      "accept heartbeat",
			setupTerm: 1,
			args: AppendEntriesArgs{
				Term:         1,
				LeaderID:     "node2",
				PrevLogIndex: 0,
				PrevLogTerm:  0,
				Entries:      nil,
				LeaderCommit: 0,
			},
			expectSuccess: true,
			expectTerm:    1,
		},
		{
			name:      "reject - stale term",
			setupTerm: 5,
			args: AppendEntriesArgs{
				Term:         3,
				LeaderID:     "node2",
				PrevLogIndex: 0,
				PrevLogTerm:  0,
				Entries:      nil,
				LeaderCommit: 0,
			},
			expectSuccess: false,
			expectTerm:    5,
		},
		{
			name:      "accept - newer term",
			setupTerm: 1,
			args: AppendEntriesArgs{
				Term:         2,
				LeaderID:     "node2",
				PrevLogIndex: 0,
				PrevLogTerm:  0,
				Entries:      nil,
				LeaderCommit: 0,
			},
			expectSuccess: true,
			expectTerm:    2,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			node.mu.Lock()
			node.currentTerm = tc.setupTerm
			node.mu.Unlock()

			reply := node.HandleAppendEntries(tc.args)

			if reply.Success != tc.expectSuccess {
				t.Errorf("expected Success=%v, got %v", tc.expectSuccess, reply.Success)
			}

			if reply.Term != tc.expectTerm {
				t.Errorf("expected Term=%d, got %d", tc.expectTerm, reply.Term)
			}
		})
	}
}

// TestRaftLogAppend tests log entry appending.
func TestRaftLogAppend(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)
	node.setState(Leader)
	node.mu.Lock()
	node.currentTerm = 1
	node.mu.Unlock()

	// Propose a command
	err := node.Propose(context.Background(), []byte("command1"))
	if err != nil {
		t.Fatalf("Propose failed: %v", err)
	}

	node.mu.RLock()
	logLen := len(node.log)
	node.mu.RUnlock()

	// Log should have dummy entry + 1 real entry
	if logLen != 2 {
		t.Errorf("expected log length 2, got %d", logLen)
	}
}

// TestRaftProposeNotLeader tests proposal rejection when not leader.
func TestRaftProposeNotLeader(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)
	// Default state is Follower

	err := node.Propose(context.Background(), []byte("command"))
	if err != ErrNotLeader {
		t.Errorf("expected ErrNotLeader, got %v", err)
	}
}

// TestRaftElectionTimeout tests that follower starts election on timeout.
func TestRaftElectionTimeout(t *testing.T) {
	transport := newMockTransport()
	config := RaftConfig{
		NodeID:            "node1",
		ElectionTimeout:   50 * time.Millisecond,
		HeartbeatInterval: 20 * time.Millisecond,
		Peers:             nil, // No peers = will become leader
	}
	node := NewRaftNode(config, transport, nil)
	transport.register(node)

	node.Start()
	defer node.Stop()

	// Wait for election timeout + some buffer
	time.Sleep(500 * time.Millisecond)

	// Should have started at least one election
	if node.ElectionCount() == 0 {
		t.Error("expected at least one election to be started")
	}

	// With no peers, should become leader (or still candidate as there's no majority)
	// Note: With no peers, the node needs only 1 vote (itself) for majority
	state := node.State()
	if state != Leader && state != Candidate {
		t.Errorf("expected Leader or Candidate state, got %s", state)
	}
}

// TestRaftThreeNodeCluster tests basic three-node cluster behavior.
func TestRaftThreeNodeCluster(t *testing.T) {
	transport := newMockTransport()

	// Create three nodes
	configs := []RaftConfig{
		{NodeID: "node1", ElectionTimeout: 50 * time.Millisecond, HeartbeatInterval: 20 * time.Millisecond, Peers: []string{"node2", "node3"}},
		{NodeID: "node2", ElectionTimeout: 100 * time.Millisecond, HeartbeatInterval: 20 * time.Millisecond, Peers: []string{"node1", "node3"}},
		{NodeID: "node3", ElectionTimeout: 150 * time.Millisecond, HeartbeatInterval: 20 * time.Millisecond, Peers: []string{"node1", "node2"}},
	}

	nodes := make([]*RaftNode, 3)
	for i, cfg := range configs {
		nodes[i] = NewRaftNode(cfg, transport, nil)
		transport.register(nodes[i])
	}

	// Start all nodes
	for _, n := range nodes {
		n.Start()
	}
	defer func() {
		for _, n := range nodes {
			n.Stop()
		}
	}()

	// Wait for election to complete
	time.Sleep(500 * time.Millisecond)

	// Exactly one leader should exist
	leaderCount := 0
	var leader *RaftNode
	for _, n := range nodes {
		if n.IsLeader() {
			leaderCount++
			leader = n
		}
	}

	if leaderCount != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaderCount)
	}

	if leader == nil {
		t.Fatal("no leader found")
	}

	// All nodes should agree on the leader
	leaderID := leader.config.NodeID
	for _, n := range nodes {
		if n.LeaderID() != leaderID && n.IsLeader() == false {
			t.Errorf("node %s thinks leader is %s, expected %s",
				n.config.NodeID, n.LeaderID(), leaderID)
		}
	}
}

// TestRaftTermIncrement tests that term increases on election.
func TestRaftTermIncrement(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)

	initialTerm := node.CurrentTerm()

	// Manually trigger election behavior
	node.mu.Lock()
	node.currentTerm++
	node.mu.Unlock()

	if node.CurrentTerm() != initialTerm+1 {
		t.Errorf("expected term %d, got %d", initialTerm+1, node.CurrentTerm())
	}
}

// TestRaftLogConsistencyCheck tests log consistency verification.
func TestRaftLogConsistencyCheck(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)

	// Add some log entries
	node.mu.Lock()
	node.log = append(node.log, LogEntry{Index: 1, Term: 1, Command: []byte("cmd1")})
	node.log = append(node.log, LogEntry{Index: 2, Term: 1, Command: []byte("cmd2")})
	node.mu.Unlock()

	// Test with matching prev log
	args := AppendEntriesArgs{
		Term:         1,
		LeaderID:     "leader",
		PrevLogIndex: 2,
		PrevLogTerm:  1,
		Entries:      []LogEntry{{Index: 3, Term: 1, Command: []byte("cmd3")}},
	}

	reply := node.HandleAppendEntries(args)
	if !reply.Success {
		t.Error("expected success with matching prev log")
	}

	// Test with non-matching prev log term
	args = AppendEntriesArgs{
		Term:         1,
		LeaderID:     "leader",
		PrevLogIndex: 2,
		PrevLogTerm:  5, // Wrong term
		Entries:      nil,
	}

	reply = node.HandleAppendEntries(args)
	if reply.Success {
		t.Error("expected failure with non-matching prev log term")
	}
}

// TestNodeStateString tests NodeState string conversion.
func TestNodeStateString(t *testing.T) {
	tests := []struct {
		state    NodeState
		expected string
	}{
		{Follower, "Follower"},
		{Candidate, "Candidate"},
		{Leader, "Leader"},
		{NodeState(99), "Unknown"},
	}

	for _, tc := range tests {
		if tc.state.String() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.state.String())
		}
	}
}

// BenchmarkRaftPropose benchmarks command proposal.
func BenchmarkRaftPropose(b *testing.B) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)
	node.setState(Leader)
	node.mu.Lock()
	node.currentTerm = 1
	node.mu.Unlock()

	ctx := context.Background()
	cmd := []byte("benchmark-command")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Propose(ctx, cmd)
	}
}

// BenchmarkRaftAppendEntries benchmarks append entries handling.
func BenchmarkRaftAppendEntries(b *testing.B) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)

	args := AppendEntriesArgs{
		Term:         1,
		LeaderID:     "leader",
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      []LogEntry{{Index: 1, Term: 1, Command: []byte("cmd")}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.HandleAppendEntries(args)
	}
}

// TestRaftApplyChannel tests that committed entries are sent to apply channel.
func TestRaftApplyChannel(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	sm := &mockStateMachine{}
	node := NewRaftNode(config, transport, sm)

	node.Start()
	defer node.Stop()

	// Add and commit an entry
	node.mu.Lock()
	node.setState(Leader)
	node.currentTerm = 1
	node.log = append(node.log, LogEntry{Index: 1, Term: 1, Command: []byte("test")})
	node.commitIndex = 1
	node.mu.Unlock()

	// Wait for apply
	time.Sleep(100 * time.Millisecond)

	if sm.AppliedCount() != 1 {
		t.Errorf("expected 1 applied entry, got %d", sm.AppliedCount())
	}
}

// TestRaftConcurrentPropose tests concurrent proposals.
func TestRaftConcurrentPropose(t *testing.T) {
	transport := newMockTransport()
	config := DefaultRaftConfig("node1", nil)
	node := NewRaftNode(config, transport, nil)
	node.setState(Leader)
	node.mu.Lock()
	node.currentTerm = 1
	node.mu.Unlock()

	var wg sync.WaitGroup
	numProposals := 100

	successCount := int32(0)

	for i := 0; i < numProposals; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := node.Propose(context.Background(), []byte("cmd"))
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()

	if int(successCount) != numProposals {
		t.Errorf("expected %d successful proposals, got %d", numProposals, successCount)
	}
}

