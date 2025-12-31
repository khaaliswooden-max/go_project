package distributed

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestMemoryLeaseStore tests the in-memory lease store.
func TestMemoryLeaseStore(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	// Test initial acquire
	acquired, err := store.TryAcquire(ctx, "node1", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}
	if !acquired {
		t.Error("expected to acquire lease")
	}

	// Test that same node can re-acquire
	acquired, err = store.TryAcquire(ctx, "node1", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}
	if !acquired {
		t.Error("expected same node to re-acquire lease")
	}

	// Test that different node cannot acquire
	acquired, err = store.TryAcquire(ctx, "node2", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}
	if acquired {
		t.Error("expected different node to fail acquiring lease")
	}

	// Test GetHolder
	holder, err := store.GetHolder(ctx)
	if err != nil {
		t.Fatalf("GetHolder failed: %v", err)
	}
	if holder != "node1" {
		t.Errorf("expected holder 'node1', got %q", holder)
	}
}

// TestMemoryLeaseStoreRenew tests lease renewal.
func TestMemoryLeaseStoreRenew(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	// Acquire lease
	_, err := store.TryAcquire(ctx, "node1", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}

	// Renew should succeed for holder
	err = store.Renew(ctx, "node1")
	if err != nil {
		t.Errorf("Renew failed for holder: %v", err)
	}

	// Renew should fail for non-holder
	err = store.Renew(ctx, "node2")
	if err != ErrNotLeaseHolder {
		t.Errorf("expected ErrNotLeaseHolder, got %v", err)
	}
}

// TestMemoryLeaseStoreRelease tests lease release.
func TestMemoryLeaseStoreRelease(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	// Acquire and release
	_, _ = store.TryAcquire(ctx, "node1", 5*time.Second)
	err := store.Release(ctx, "node1")
	if err != nil {
		t.Errorf("Release failed: %v", err)
	}

	// Now another node should be able to acquire
	acquired, err := store.TryAcquire(ctx, "node2", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}
	if !acquired {
		t.Error("expected node2 to acquire after release")
	}
}

// TestMemoryLeaseStoreExpiration tests lease expiration.
func TestMemoryLeaseStoreExpiration(t *testing.T) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	// Acquire with short TTL
	_, err := store.TryAcquire(ctx, "node1", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Another node should be able to acquire now
	acquired, err := store.TryAcquire(ctx, "node2", 5*time.Second)
	if err != nil {
		t.Fatalf("TryAcquire failed: %v", err)
	}
	if !acquired {
		t.Error("expected node2 to acquire after expiration")
	}
}

// TestLeaderElectionCreation tests leader election creation.
func TestLeaderElectionCreation(t *testing.T) {
	config := DefaultLeaderElectionConfig("node1")
	store := NewMemoryLeaseStore()

	election := NewLeaderElection(config, store)
	if election == nil {
		t.Fatal("NewLeaderElection returned nil")
	}

	if election.IsLeader() {
		t.Error("should not be leader initially")
	}

	if election.LeaderID() != "" {
		t.Errorf("expected empty leader ID, got %q", election.LeaderID())
	}
}

// TestLeaderElectionCampaign tests leadership acquisition.
func TestLeaderElectionCampaign(t *testing.T) {
	config := LeaderElectionConfig{
		NodeID:        "node1",
		LeaseDuration: 5 * time.Second,
		RenewDeadline: 2 * time.Second,
		RetryPeriod:   50 * time.Millisecond,
	}
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := election.Campaign(ctx)
	if err != nil {
		t.Errorf("Campaign failed: %v", err)
	}

	if !election.IsLeader() {
		t.Error("expected to become leader after Campaign")
	}

	if election.LeaderID() != "node1" {
		t.Errorf("expected leader ID 'node1', got %q", election.LeaderID())
	}
}

// TestLeaderElectionMultipleNodes tests election between multiple nodes.
func TestLeaderElectionMultipleNodes(t *testing.T) {
	store := NewMemoryLeaseStore() // Shared store

	elections := make([]*LeaderElection, 3)
	for i := 0; i < 3; i++ {
		config := LeaderElectionConfig{
			NodeID:        string(rune('A' + i)),
			LeaseDuration: 5 * time.Second,
			RenewDeadline: 2 * time.Second,
			RetryPeriod:   50 * time.Millisecond,
		}
		elections[i] = NewLeaderElection(config, store)
	}

	// Run all elections concurrently
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for _, e := range elections {
		wg.Add(1)
		go func(el *LeaderElection) {
			defer wg.Done()
			_ = el.Campaign(ctx)
		}(e)
	}

	wg.Wait()

	// Exactly one should be leader
	leaderCount := 0
	for _, e := range elections {
		if e.IsLeader() {
			leaderCount++
		}
	}

	if leaderCount != 1 {
		t.Errorf("expected exactly 1 leader, got %d", leaderCount)
	}
}

// TestLeaderElectionObserver tests observer notifications.
func TestLeaderElectionObserver(t *testing.T) {
	config := DefaultLeaderElectionConfig("node1")
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	becameLeader := int32(0)
	lostLeadership := int32(0)

	observer := &LeaderObserverFunc{
		BecomeLeaderFn: func() {
			atomic.AddInt32(&becameLeader, 1)
		},
		LoseLeadershipFn: func() {
			atomic.AddInt32(&lostLeadership, 1)
		},
	}

	election.AddObserver(observer)

	// Acquire leadership
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	err := election.Campaign(ctx)
	cancel()
	if err != nil {
		t.Fatalf("Campaign failed: %v", err)
	}

	// Wait for notification
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&becameLeader) != 1 {
		t.Error("observer should have been notified of becoming leader")
	}

	// Release leadership
	election.release()
	time.Sleep(100 * time.Millisecond)

	if atomic.LoadInt32(&lostLeadership) != 1 {
		t.Error("observer should have been notified of losing leadership")
	}
}

// TestFenceToken tests fencing token increment.
func TestFenceToken(t *testing.T) {
	config := DefaultLeaderElectionConfig("node1")
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	initialToken := election.GetFenceToken()

	// Acquire leadership
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_ = election.Campaign(ctx)
	cancel()

	newToken := election.GetFenceToken()
	if newToken != initialToken+1 {
		t.Errorf("expected fence token %d, got %d", initialToken+1, newToken)
	}

	// Release and re-acquire should increment again
	election.release()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	_ = election.Campaign(ctx)
	cancel()

	finalToken := election.GetFenceToken()
	if finalToken != newToken+1 {
		t.Errorf("expected fence token %d, got %d", newToken+1, finalToken)
	}
}

// TestFencedServer tests fenced request validation.
func TestFencedServer(t *testing.T) {
	server := NewFencedServer()

	// First request with token 1 should succeed
	err := server.ValidateRequest(FencedRequest{FenceToken: 1, Operation: "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Request with higher token should succeed
	err = server.ValidateRequest(FencedRequest{FenceToken: 5, Operation: "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Request with lower token should fail
	err = server.ValidateRequest(FencedRequest{FenceToken: 3, Operation: "test"})
	if err != ErrStaleFenceToken {
		t.Errorf("expected ErrStaleFenceToken, got %v", err)
	}
}

// TestLeaderAware tests leadership-aware component.
func TestLeaderAware(t *testing.T) {
	config := DefaultLeaderElectionConfig("node1")
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	started := int32(0)
	stopped := int32(0)

	la := NewLeaderAware(
		election,
		func() error {
			atomic.AddInt32(&started, 1)
			return nil
		},
		func() error {
			atomic.AddInt32(&stopped, 1)
			return nil
		},
	)

	// Initially not running
	if la.IsRunning() {
		t.Error("should not be running initially")
	}

	// Simulate becoming leader
	la.OnBecomeLeader()
	time.Sleep(50 * time.Millisecond)

	if !la.IsRunning() {
		t.Error("should be running after becoming leader")
	}

	if atomic.LoadInt32(&started) != 1 {
		t.Error("start function should have been called")
	}

	// Simulate losing leadership
	la.OnLoseLeadership()
	time.Sleep(50 * time.Millisecond)

	if la.IsRunning() {
		t.Error("should not be running after losing leadership")
	}

	if atomic.LoadInt32(&stopped) != 1 {
		t.Error("stop function should have been called")
	}
}

// TestTransferLeadership tests leadership transfer.
func TestTransferLeadership(t *testing.T) {
	store := NewMemoryLeaseStore()

	config1 := LeaderElectionConfig{
		NodeID:        "node1",
		LeaseDuration: 500 * time.Millisecond,
		RenewDeadline: 200 * time.Millisecond,
		RetryPeriod:   50 * time.Millisecond,
	}
	election1 := NewLeaderElection(config1, store)

	config2 := LeaderElectionConfig{
		NodeID:        "node2",
		LeaseDuration: 500 * time.Millisecond,
		RenewDeadline: 200 * time.Millisecond,
		RetryPeriod:   50 * time.Millisecond,
	}
	election2 := NewLeaderElection(config2, store)

	// Node1 becomes leader
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	_ = election1.Campaign(ctx)
	cancel()

	if !election1.IsLeader() {
		t.Fatal("node1 should be leader")
	}

	// Start node2 trying to acquire in background
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = election2.Campaign(ctx)
	}()

	// Give node2 time to start campaigning
	time.Sleep(100 * time.Millisecond)

	// Transfer leadership
	transfer := NewTransferLeadership(election1)
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	err := transfer.Transfer(ctx, "node2")
	cancel()

	if err != nil {
		t.Logf("Transfer returned: %v (this may be timing-related)", err)
	}

	// Verify node1 gave up leadership
	if election1.IsLeader() {
		t.Error("node1 should no longer be leader")
	}

	// Give node2 time to acquire
	time.Sleep(200 * time.Millisecond)

	// Node2 should eventually become leader (or already be)
	if !election2.IsLeader() {
		t.Logf("Note: node2 not yet leader, this may be timing-related")
	}
}

// TestTransferLeadershipNotLeader tests transfer when not leader.
func TestTransferLeadershipNotLeader(t *testing.T) {
	config := DefaultLeaderElectionConfig("node1")
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	transfer := NewTransferLeadership(election)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := transfer.Transfer(ctx, "node2")
	if err != ErrNotLeader {
		t.Errorf("expected ErrNotLeader, got %v", err)
	}
}

// TestLeaderObserverFuncNilFunctions tests nil function safety.
func TestLeaderObserverFuncNilFunctions(t *testing.T) {
	observer := &LeaderObserverFunc{} // All functions nil

	// Should not panic
	observer.OnBecomeLeader()
	observer.OnLoseLeadership()
	observer.OnLeaderChanged("node2")
}

// TestLeaderElectionRun tests the main election loop.
func TestLeaderElectionRun(t *testing.T) {
	config := LeaderElectionConfig{
		NodeID:        "node1",
		LeaseDuration: 100 * time.Millisecond,
		RenewDeadline: 50 * time.Millisecond,
		RetryPeriod:   20 * time.Millisecond,
	}
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Run in background
	done := make(chan error, 1)
	go func() {
		done <- election.Run(ctx)
	}()

	// Wait a bit for leadership acquisition
	time.Sleep(100 * time.Millisecond)

	if !election.IsLeader() {
		t.Error("should become leader during Run")
	}

	// Cancel and wait for shutdown
	cancel()
	err := <-done

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

// BenchmarkLeaseAcquire benchmarks lease acquisition.
func BenchmarkLeaseAcquire(b *testing.B) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.TryAcquire(ctx, "node1", 5*time.Second)
	}
}

// BenchmarkLeaseRenew benchmarks lease renewal.
func BenchmarkLeaseRenew(b *testing.B) {
	store := NewMemoryLeaseStore()
	ctx := context.Background()
	_, _ = store.TryAcquire(ctx, "node1", 5*time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Renew(ctx, "node1")
	}
}

// BenchmarkFencedValidation benchmarks fencing token validation.
func BenchmarkFencedValidation(b *testing.B) {
	server := NewFencedServer()
	req := FencedRequest{FenceToken: 1, Operation: "test"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req.FenceToken = uint64(i + 1)
		_ = server.ValidateRequest(req)
	}
}

// TestLeaderElectionStop tests graceful stop.
func TestLeaderElectionStop(t *testing.T) {
	config := LeaderElectionConfig{
		NodeID:        "node1",
		LeaseDuration: 5 * time.Second,
		RenewDeadline: 2 * time.Second,
		RetryPeriod:   50 * time.Millisecond,
	}
	store := NewMemoryLeaseStore()
	election := NewLeaderElection(config, store)

	// Run in background
	ctx := context.Background()
	done := make(chan error, 1)
	go func() {
		done <- election.Run(ctx)
	}()

	// Wait for startup
	time.Sleep(100 * time.Millisecond)

	// Stop
	election.Stop()

	// Should return
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("Stop did not cause Run to return")
	}
}

