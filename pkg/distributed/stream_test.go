package distributed

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestChannelStreamBasic tests basic send/receive operations.
func TestChannelStreamBasic(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[string](ctx, 10)

	// Sender goroutine
	go func() {
		defer stream.Close()
		for i := 0; i < 5; i++ {
			if err := stream.Send("message"); err != nil {
				t.Errorf("Send failed: %v", err)
			}
		}
	}()

	// Receive messages from send channel
	count := 0
	for msg := range stream.SendCh() {
		if msg != "message" {
			t.Errorf("expected 'message', got %q", msg)
		}
		count++
	}

	if count != 5 {
		t.Errorf("expected 5 messages, got %d", count)
	}
}

// TestChannelStreamContext tests context cancellation.
func TestChannelStreamContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	stream := NewChannelStream[string](ctx, 0) // Unbuffered

	// Cancel immediately
	cancel()

	// Send should fail with context error
	err := stream.Send("test")
	if err == nil {
		t.Error("expected error from cancelled context")
	}

	// Recv should also fail
	_, err = stream.Recv()
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}

// TestChannelStreamClose tests stream closure.
func TestChannelStreamClose(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, 10)

	// Close the stream
	if err := stream.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Send should fail after close
	err := stream.Send(42)
	if err != ErrStreamClosed {
		t.Errorf("expected ErrStreamClosed, got %v", err)
	}

	// Double close should be safe
	if err := stream.Close(); err != nil {
		t.Errorf("double Close failed: %v", err)
	}
}

// TestStreamPairCommunication tests bidirectional communication.
func TestStreamPairCommunication(t *testing.T) {
	ctx := context.Background()
	pair := NewStreamPair[string](ctx, 10)

	var wg sync.WaitGroup

	// Client sends, server receives
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 3; i++ {
			if err := pair.Client.Send("from-client"); err != nil {
				t.Errorf("client send failed: %v", err)
			}
		}
		pair.Client.Close()
	}()

	go func() {
		defer wg.Done()
		count := 0
		for {
			msg, err := pair.Server.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Errorf("server recv failed: %v", err)
				break
			}
			if msg != "from-client" {
				t.Errorf("expected 'from-client', got %q", msg)
			}
			count++
		}
		if count != 3 {
			t.Errorf("expected 3 messages, got %d", count)
		}
	}()

	wg.Wait()
}

// TestFlowControlledStream tests backpressure mechanism.
func TestFlowControlledStream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	baseStream := NewChannelStream[int](ctx, 0) // Unbuffered
	stream := NewFlowControlledStream[int](baseStream, 3)

	// Track pending count
	if stream.Pending() != 0 {
		t.Errorf("expected 0 pending, got %d", stream.Pending())
	}

	// Start a consumer
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			select {
			case <-baseStream.SendCh():
				stream.Ack()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Send messages
	for i := 0; i < 5; i++ {
		if err := stream.Send(i); err != nil {
			t.Errorf("Send failed: %v", err)
		}
	}

	wg.Wait()
}

// TestServerStream tests server streaming pattern.
func TestServerStream(t *testing.T) {
	ctx := context.Background()

	// Handler that sends multiple responses
	handler := func(req string, stream Stream[string]) error {
		for i := 0; i < 5; i++ {
			if err := stream.Send(req + "-response"); err != nil {
				return err
			}
		}
		return nil
	}

	respCh, errCh := RunServerStream(ctx, "request", handler, 10)

	// Collect responses
	var responses []string
	for resp := range respCh {
		responses = append(responses, resp)
	}

	// Check for errors
	if err := <-errCh; err != nil {
		t.Errorf("handler returned error: %v", err)
	}

	if len(responses) != 5 {
		t.Errorf("expected 5 responses, got %d", len(responses))
	}

	for _, resp := range responses {
		if resp != "request-response" {
			t.Errorf("unexpected response: %s", resp)
		}
	}
}

// TestClientStream tests client streaming pattern.
func TestClientStream(t *testing.T) {
	ctx := context.Background()

	// Handler that collects requests and returns count
	handler := func(stream Stream[int]) (int, error) {
		count := 0
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return 0, err
			}
			count++
		}
		return count, nil
	}

	reqCh, respCh, errCh := RunClientStream(ctx, handler, 10)

	// Send requests
	for i := 0; i < 10; i++ {
		reqCh <- i
	}
	close(reqCh)

	// Get response
	resp := <-respCh
	if resp != 10 {
		t.Errorf("expected count 10, got %d", resp)
	}

	// Check for errors
	if err := <-errCh; err != nil {
		t.Errorf("handler returned error: %v", err)
	}
}

// TestBidiStreamPair tests bidirectional stream pair creation.
func TestBidiStreamPair(t *testing.T) {
	ctx := context.Background()
	pair := NewBidiStreamPair[string, int](ctx, 10)

	if pair.ReqStream == nil {
		t.Error("ReqStream is nil")
	}
	if pair.RespStream == nil {
		t.Error("RespStream is nil")
	}

	// Test that streams are independent
	pair.ReqStream.Send("request")
	pair.RespStream.Send(42)
}

// TestCollectStream tests collecting all messages from a stream.
func TestCollectStream(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, 10)

	// Send some values
	go func() {
		for i := 1; i <= 5; i++ {
			stream.Send(i)
		}
		stream.Close() // Close the send channel
	}()

	// Collect using the send channel (which we read from)
	recvStream := &recvOnlyStream[int]{
		ch:  stream.SendCh(),
		ctx: ctx,
	}

	values, err := CollectStream[int](recvStream)
	if err != nil {
		t.Errorf("CollectStream failed: %v", err)
	}

	expected := []int{1, 2, 3, 4, 5}
	if len(values) != len(expected) {
		t.Errorf("expected %d values, got %d", len(expected), len(values))
	}

	for i, v := range values {
		if v != expected[i] {
			t.Errorf("expected values[%d]=%d, got %d", i, expected[i], v)
		}
	}
}

// TestSendAll tests sending all items to a stream.
func TestSendAll(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[string](ctx, 10)

	items := []string{"a", "b", "c"}

	go func() {
		if err := SendAll[string](stream, items); err != nil {
			t.Errorf("SendAll failed: %v", err)
		}
		stream.Close()
	}()

	// Receive all
	var received []string
	for msg := range stream.SendCh() {
		received = append(received, msg)
	}

	if len(received) != len(items) {
		t.Errorf("expected %d items, got %d", len(items), len(received))
	}
}

// TestTimeoutStream tests stream operation timeouts.
func TestTimeoutStream(t *testing.T) {
	ctx := context.Background()
	baseStream := NewChannelStream[string](ctx, 0) // Unbuffered

	stream := NewTimeoutStream(baseStream, 50*time.Millisecond, 50*time.Millisecond)

	// Send should timeout (no receiver)
	err := stream.Send("test")
	if err == nil {
		t.Error("expected timeout error on send")
	}

	// Recv should timeout (no sender)
	_, err = stream.Recv()
	if err == nil {
		t.Error("expected timeout error on recv")
	}
}

// TestTimeoutStreamSuccess tests successful operations within timeout.
func TestTimeoutStreamSuccess(t *testing.T) {
	ctx := context.Background()
	baseStream := NewChannelStream[int](ctx, 1) // Buffered

	stream := NewTimeoutStream(baseStream, time.Second, time.Second)

	// Send should succeed (buffered)
	err := stream.Send(42)
	if err != nil {
		t.Errorf("expected successful send, got %v", err)
	}
}

// TestStreamTypeString tests StreamType string conversion.
func TestStreamTypeString(t *testing.T) {
	tests := []struct {
		st       StreamType
		expected string
	}{
		{UnaryStream, "Unary"},
		{ServerStream, "ServerStream"},
		{ClientStream, "ClientStream"},
		{BidirectionalStream, "Bidirectional"},
		{StreamType(99), "Unknown"},
	}

	for _, tc := range tests {
		if tc.st.String() != tc.expected {
			t.Errorf("expected %s, got %s", tc.expected, tc.st.String())
		}
	}
}

// TestConcurrentStreamOperations tests thread safety.
func TestConcurrentStreamOperations(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream := NewChannelStream[int](ctx, 100)

	var wg sync.WaitGroup
	numSenders := 10
	numMessages := 100

	// Multiple senders
	for i := 0; i < numSenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < numMessages; j++ {
				if err := stream.Send(j); err != nil {
					return
				}
			}
		}()
	}

	// Receiver counts messages
	received := int32(0)
	done := make(chan struct{})

	go func() {
		for range stream.SendCh() {
			atomic.AddInt32(&received, 1)
		}
		close(done)
	}()

	wg.Wait()
	stream.Close()
	<-done

	expected := int32(numSenders * numMessages)
	if atomic.LoadInt32(&received) != expected {
		t.Errorf("expected %d messages, got %d", expected, received)
	}
}

// BenchmarkStreamSend benchmarks stream send operations.
func BenchmarkStreamSend(b *testing.B) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, b.N)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stream.Send(i)
	}
}

// BenchmarkStreamSendRecv benchmarks send/receive pairs.
func BenchmarkStreamSendRecv(b *testing.B) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, 100)

	b.ResetTimer()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < b.N; i++ {
			_ = stream.Send(i)
		}
		stream.Close()
	}()

	go func() {
		defer wg.Done()
		for range stream.SendCh() {
		}
	}()

	wg.Wait()
}

// BenchmarkFlowControlledStream benchmarks flow controlled operations.
func BenchmarkFlowControlledStream(b *testing.B) {
	ctx := context.Background()
	baseStream := NewChannelStream[int](ctx, 100)
	stream := NewFlowControlledStream[int](baseStream, 50)

	// Consumer
	go func() {
		for range baseStream.SendCh() {
			stream.Ack()
		}
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stream.Send(i)
	}

	stream.Close()
}

// TestRecvOnlyStreamSendError tests that send fails on recv-only stream.
func TestRecvOnlyStreamSendError(t *testing.T) {
	stream := &recvOnlyStream[int]{
		ch:  make(chan int),
		ctx: context.Background(),
	}

	err := stream.Send(42)
	if err == nil {
		t.Error("expected error on send to recv-only stream")
	}
}

// TestSendOnlyStreamRecvError tests that recv fails on send-only stream.
func TestSendOnlyStreamRecvError(t *testing.T) {
	stream := &sendOnlyStream[int]{
		ch:  make(chan int),
		ctx: context.Background(),
	}

	_, err := stream.Recv()
	if err == nil {
		t.Error("expected error on recv from send-only stream")
	}
}

// TestStreamCancel tests stream cancellation.
func TestStreamCancel(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, 0)

	stream.Cancel()

	// Operations should fail after cancel
	err := stream.Send(42)
	if err == nil {
		t.Error("expected error after cancel")
	}
}

// TestCloseRecvSafety tests that CloseRecv is safe to call multiple times.
func TestCloseRecvSafety(t *testing.T) {
	ctx := context.Background()
	stream := NewChannelStream[int](ctx, 10)

	// Should not panic
	stream.CloseRecv()
	stream.CloseRecv() // Second call should be safe
}

