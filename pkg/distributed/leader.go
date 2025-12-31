package distributed

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// LEARN: Leader election patterns
// Simpler than full Raft when you only need coordination.

// LeaderObserver receives notifications about leadership changes.
type LeaderObserver interface {
	// OnBecomeLeader is called when this node becomes leader.
	OnBecomeLeader()

	// OnLoseLeadership is called when this node loses leadership.
	OnLoseLeadership()

	// OnLeaderChanged is called when the leader changes.
	OnLeaderChanged(newLeader string)
}

// LEARN: Lease-based leader election
// Leader holds a lease that must be renewed periodically.
// If renewal fails, another node can acquire leadership.

// LeaseStore is the interface for distributed lease storage.
type LeaseStore interface {
	// TryAcquire attempts to acquire the lease.
	// Returns true if lease was acquired, false if already held by another.
	TryAcquire(ctx context.Context, nodeID string, ttl time.Duration) (bool, error)

	// Renew extends the lease for this node.
	// Returns error if lease expired or held by another node.
	Renew(ctx context.Context, nodeID string) error

	// Release voluntarily releases the lease.
	Release(ctx context.Context, nodeID string) error

	// GetHolder returns the current lease holder.
	GetHolder(ctx context.Context) (string, error)
}

// LeaderElectionConfig configures leader election.
type LeaderElectionConfig struct {
	// NodeID uniquely identifies this node.
	NodeID string

	// LeaseDuration is how long the lease is valid.
	LeaseDuration time.Duration

	// RenewDeadline is the maximum time for renewal.
	// Should be less than LeaseDuration.
	RenewDeadline time.Duration

	// RetryPeriod is how often to retry acquiring leadership.
	RetryPeriod time.Duration
}

// DefaultLeaderElectionConfig returns sensible defaults.
func DefaultLeaderElectionConfig(nodeID string) LeaderElectionConfig {
	return LeaderElectionConfig{
		NodeID:        nodeID,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}
}

// LeaderElection manages lease-based leader election.
type LeaderElection struct {
	config    LeaderElectionConfig
	store     LeaseStore
	observers []LeaderObserver

	isLeader   int32
	leaderID   atomic.Value
	fenceToken uint64

	renewCtx  context.Context
	renewStop context.CancelFunc

	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewLeaderElection creates a new leader election instance.
func NewLeaderElection(config LeaderElectionConfig, store LeaseStore) *LeaderElection {
	le := &LeaderElection{
		config:   config,
		store:    store,
		stopCh:   make(chan struct{}),
	}
	le.leaderID.Store("")
	return le
}

// AddObserver registers an observer for leadership changes.
func (le *LeaderElection) AddObserver(o LeaderObserver) {
	le.mu.Lock()
	defer le.mu.Unlock()
	le.observers = append(le.observers, o)
}

// Run starts the leader election process.
// Blocks until context is cancelled.
func (le *LeaderElection) Run(ctx context.Context) error {
	ticker := time.NewTicker(le.config.RetryPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			le.release()
			return ctx.Err()
		case <-le.stopCh:
			le.release()
			return nil
		case <-ticker.C:
			if le.IsLeader() {
				if err := le.renew(ctx); err != nil {
					// Lost leadership
					le.loseLeadership()
				}
			} else {
				le.tryAcquire(ctx)
			}
		}
	}
}

// Campaign attempts to become leader.
// Blocks until leadership is acquired or context is cancelled.
func (le *LeaderElection) Campaign(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		acquired, err := le.store.TryAcquire(ctx, le.config.NodeID, le.config.LeaseDuration)
		if err != nil {
			time.Sleep(le.config.RetryPeriod)
			continue
		}

		if acquired {
			le.becomeLeader()
			return nil
		}

		time.Sleep(le.config.RetryPeriod)
	}
}

// Stop gracefully stops the leader election.
func (le *LeaderElection) Stop() {
	close(le.stopCh)
}

// IsLeader returns true if this node is the current leader.
func (le *LeaderElection) IsLeader() bool {
	return atomic.LoadInt32(&le.isLeader) == 1
}

// LeaderID returns the current leader's ID.
func (le *LeaderElection) LeaderID() string {
	if v := le.leaderID.Load(); v != nil {
		return v.(string)
	}
	return ""
}

// GetFenceToken returns the current fencing token.
// LEARN: Fencing tokens prevent stale leaders from corrupting state.
func (le *LeaderElection) GetFenceToken() uint64 {
	le.mu.RLock()
	defer le.mu.RUnlock()
	return le.fenceToken
}

func (le *LeaderElection) tryAcquire(ctx context.Context) {
	acquired, err := le.store.TryAcquire(ctx, le.config.NodeID, le.config.LeaseDuration)
	if err != nil {
		return
	}

	if acquired {
		le.becomeLeader()
	} else {
		// Check who the leader is
		holder, err := le.store.GetHolder(ctx)
		if err == nil && holder != le.LeaderID() {
			le.notifyLeaderChanged(holder)
		}
	}
}

func (le *LeaderElection) renew(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, le.config.RenewDeadline)
	defer cancel()

	return le.store.Renew(ctx, le.config.NodeID)
}

func (le *LeaderElection) release() {
	if le.IsLeader() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = le.store.Release(ctx, le.config.NodeID)
		le.loseLeadership()
	}
}

func (le *LeaderElection) becomeLeader() {
	le.mu.Lock()
	le.fenceToken++
	le.mu.Unlock()

	atomic.StoreInt32(&le.isLeader, 1)
	le.leaderID.Store(le.config.NodeID)

	// Start renewal loop
	le.renewCtx, le.renewStop = context.WithCancel(context.Background())

	le.notifyBecomeLeader()
}

func (le *LeaderElection) loseLeadership() {
	if atomic.CompareAndSwapInt32(&le.isLeader, 1, 0) {
		if le.renewStop != nil {
			le.renewStop()
		}
		le.leaderID.Store("")
		le.notifyLoseLeadership()
	}
}

func (le *LeaderElection) notifyBecomeLeader() {
	le.mu.RLock()
	observers := le.observers
	le.mu.RUnlock()

	for _, o := range observers {
		go o.OnBecomeLeader()
	}
}

func (le *LeaderElection) notifyLoseLeadership() {
	le.mu.RLock()
	observers := le.observers
	le.mu.RUnlock()

	for _, o := range observers {
		go o.OnLoseLeadership()
	}
}

func (le *LeaderElection) notifyLeaderChanged(newLeader string) {
	le.leaderID.Store(newLeader)

	le.mu.RLock()
	observers := le.observers
	le.mu.RUnlock()

	for _, o := range observers {
		go o.OnLeaderChanged(newLeader)
	}
}

// LEARN: In-memory lease store for testing
// Production would use etcd, consul, or database.

// MemoryLeaseStore is an in-memory lease store for testing.
type MemoryLeaseStore struct {
	holder     string
	expiration time.Time
	mu         sync.Mutex
}

// NewMemoryLeaseStore creates a new in-memory lease store.
func NewMemoryLeaseStore() *MemoryLeaseStore {
	return &MemoryLeaseStore{}
}

// TryAcquire attempts to acquire the lease.
func (s *MemoryLeaseStore) TryAcquire(ctx context.Context, nodeID string, ttl time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	// Check if lease is expired or held by requester
	if s.holder == "" || s.holder == nodeID || now.After(s.expiration) {
		s.holder = nodeID
		s.expiration = now.Add(ttl)
		return true, nil
	}

	return false, nil
}

// Renew extends the lease.
func (s *MemoryLeaseStore) Renew(ctx context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.holder != nodeID {
		return ErrNotLeaseHolder
	}

	if time.Now().After(s.expiration) {
		return ErrLeaseExpired
	}

	// Extend by original TTL (simplified - real impl would track TTL)
	s.expiration = time.Now().Add(15 * time.Second)
	return nil
}

// Release releases the lease.
func (s *MemoryLeaseStore) Release(ctx context.Context, nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.holder == nodeID {
		s.holder = ""
		s.expiration = time.Time{}
	}
	return nil
}

// GetHolder returns the current lease holder.
func (s *MemoryLeaseStore) GetHolder(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Now().After(s.expiration) {
		return "", nil
	}
	return s.holder, nil
}

// LEARN: Fencing prevents split-brain scenarios
// Each leadership term has a unique, monotonically increasing token.

// FencedRequest includes a fencing token for safety.
type FencedRequest struct {
	FenceToken uint64
	Operation  string
	Data       []byte
}

// FencedServer validates fencing tokens on requests.
type FencedServer struct {
	lastKnownToken uint64
	mu             sync.Mutex
}

// NewFencedServer creates a new fenced server.
func NewFencedServer() *FencedServer {
	return &FencedServer{}
}

// ValidateRequest checks if the request's fence token is valid.
// Returns error if token is stale (from old leader).
func (s *FencedServer) ValidateRequest(req FencedRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if req.FenceToken < s.lastKnownToken {
		return ErrStaleFenceToken
	}

	s.lastKnownToken = req.FenceToken
	return nil
}

// LEARN: Leadership transfer for graceful handoff
// Useful for maintenance or rolling updates.

// TransferLeadership attempts to transfer leadership to target.
type TransferLeadership struct {
	election *LeaderElection
}

// NewTransferLeadership creates a leadership transfer handler.
func NewTransferLeadership(election *LeaderElection) *TransferLeadership {
	return &TransferLeadership{election: election}
}

// Transfer transfers leadership to target node.
func (t *TransferLeadership) Transfer(ctx context.Context, target string) error {
	if !t.election.IsLeader() {
		return ErrNotLeader
	}

	// Release our lease to allow target to acquire
	t.election.release()

	// Wait for target to become leader
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return ErrTransferTimeout
		case <-ticker.C:
			if t.election.LeaderID() == target {
				return nil
			}
		}
	}
}

// Errors for leader election
var (
	ErrNotLeaseHolder   = errors.New("not the lease holder")
	ErrLeaseExpired     = errors.New("lease expired")
	ErrStaleFenceToken  = errors.New("stale fence token")
	ErrTransferTimeout  = errors.New("leadership transfer timeout")
)

// LEARN: Simple observer implementation for common use cases

// LeaderObserverFunc is a function-based observer.
type LeaderObserverFunc struct {
	BecomeLeaderFn    func()
	LoseLeadershipFn  func()
	LeaderChangedFn   func(newLeader string)
}

// OnBecomeLeader implements LeaderObserver.
func (f *LeaderObserverFunc) OnBecomeLeader() {
	if f.BecomeLeaderFn != nil {
		f.BecomeLeaderFn()
	}
}

// OnLoseLeadership implements LeaderObserver.
func (f *LeaderObserverFunc) OnLoseLeadership() {
	if f.LoseLeadershipFn != nil {
		f.LoseLeadershipFn()
	}
}

// OnLeaderChanged implements LeaderObserver.
func (f *LeaderObserverFunc) OnLeaderChanged(newLeader string) {
	if f.LeaderChangedFn != nil {
		f.LeaderChangedFn(newLeader)
	}
}

// LEARN: Leadership-aware component wrapper
// Automatically enables/disables based on leadership.

// LeaderAware wraps a component that should only be active when leader.
type LeaderAware struct {
	election  *LeaderElection
	startFn   func() error
	stopFn    func() error
	isRunning int32
	mu        sync.Mutex
}

// NewLeaderAware creates a leadership-aware wrapper.
func NewLeaderAware(election *LeaderElection, startFn, stopFn func() error) *LeaderAware {
	la := &LeaderAware{
		election: election,
		startFn:  startFn,
		stopFn:   stopFn,
	}

	// Register as observer
	election.AddObserver(la)

	return la
}

// OnBecomeLeader starts the component.
func (la *LeaderAware) OnBecomeLeader() {
	la.mu.Lock()
	defer la.mu.Unlock()

	if atomic.CompareAndSwapInt32(&la.isRunning, 0, 1) {
		if la.startFn != nil {
			if err := la.startFn(); err != nil {
				atomic.StoreInt32(&la.isRunning, 0)
			}
		}
	}
}

// OnLoseLeadership stops the component.
func (la *LeaderAware) OnLoseLeadership() {
	la.mu.Lock()
	defer la.mu.Unlock()

	if atomic.CompareAndSwapInt32(&la.isRunning, 1, 0) {
		if la.stopFn != nil {
			_ = la.stopFn()
		}
	}
}

// OnLeaderChanged is a no-op for this implementation.
func (la *LeaderAware) OnLeaderChanged(newLeader string) {
	// No action needed
}

// IsRunning returns true if the component is active.
func (la *LeaderAware) IsRunning() bool {
	return atomic.LoadInt32(&la.isRunning) == 1
}

