package distributed

import (
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// LEARN: Stream types inspired by gRPC
// These patterns work regardless of transport layer.

// StreamType indicates the streaming pattern.
type StreamType int

const (
	// UnaryStream is single request, single response.
	UnaryStream StreamType = iota
	// ServerStream is single request, multiple responses.
	ServerStream
	// ClientStream is multiple requests, single response.
	ClientStream
	// BidirectionalStream is multiple requests and responses.
	BidirectionalStream
)

func (t StreamType) String() string {
	switch t {
	case UnaryStream:
		return "Unary"
	case ServerStream:
		return "ServerStream"
	case ClientStream:
		return "ClientStream"
	case BidirectionalStream:
		return "Bidirectional"
	default:
		return "Unknown"
	}
}

// LEARN: Generic stream interface
// Works with any message type using Go generics.

// Stream provides bidirectional message streaming.
type Stream[T any] interface {
	// Send sends a message to the stream.
	// Returns error if stream is closed or context is cancelled.
	Send(msg T) error

	// Recv receives a message from the stream.
	// Returns io.EOF when stream is closed by sender.
	Recv() (T, error)

	// Close closes the send side of the stream.
	Close() error

	// Context returns the stream's context.
	Context() context.Context
}

// LEARN: Channel-based stream implementation
// Uses Go channels for in-process streaming.

// ChannelStream implements Stream using channels.
type ChannelStream[T any] struct {
	sendCh chan T
	recvCh chan T
	ctx    context.Context
	cancel context.CancelFunc
	closed int32
	mu     sync.Mutex
}

// NewChannelStream creates a new bidirectional channel stream.
func NewChannelStream[T any](ctx context.Context, bufferSize int) *ChannelStream[T] {
	ctx, cancel := context.WithCancel(ctx)
	return &ChannelStream[T]{
		sendCh: make(chan T, bufferSize),
		recvCh: make(chan T, bufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Send sends a message on the stream.
func (s *ChannelStream[T]) Send(msg T) error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return ErrStreamClosed
	}

	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case s.sendCh <- msg:
		return nil
	}
}

// Recv receives a message from the stream.
func (s *ChannelStream[T]) Recv() (T, error) {
	var zero T

	select {
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	case msg, ok := <-s.recvCh:
		if !ok {
			return zero, io.EOF
		}
		return msg, nil
	}
}

// Close closes the stream's send channel.
func (s *ChannelStream[T]) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if atomic.CompareAndSwapInt32(&s.closed, 0, 1) {
		close(s.sendCh)
	}
	return nil
}

// Context returns the stream's context.
func (s *ChannelStream[T]) Context() context.Context {
	return s.ctx
}

// Cancel cancels the stream.
func (s *ChannelStream[T]) Cancel() {
	s.cancel()
}

// SendCh returns the send channel for the other side to read from.
func (s *ChannelStream[T]) SendCh() <-chan T {
	return s.sendCh
}

// RecvCh returns the receive channel for the other side to write to.
func (s *ChannelStream[T]) RecvCh() chan<- T {
	return s.recvCh
}

// CloseRecv closes the receive channel (called by sender side).
func (s *ChannelStream[T]) CloseRecv() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Use recover in case channel is already closed
	defer func() { _ = recover() }()
	close(s.recvCh)
}

// LEARN: StreamPair creates two connected streams
// One for client, one for server.

// StreamPair holds a connected pair of streams.
type StreamPair[T any] struct {
	Client *ChannelStream[T]
	Server *ChannelStream[T]
}

// NewStreamPair creates a connected client/server stream pair.
func NewStreamPair[T any](ctx context.Context, bufferSize int) *StreamPair[T] {
	clientStream := NewChannelStream[T](ctx, bufferSize)
	serverStream := NewChannelStream[T](ctx, bufferSize)

	// Cross-connect the channels
	// Client sends -> Server receives
	// Server sends -> Client receives
	pair := &StreamPair[T]{
		Client: &ChannelStream[T]{
			sendCh: clientStream.sendCh,
			recvCh: serverStream.sendCh, // Read from server's send
			ctx:    clientStream.ctx,
			cancel: clientStream.cancel,
		},
		Server: &ChannelStream[T]{
			sendCh: serverStream.sendCh,
			recvCh: clientStream.sendCh, // Read from client's send
			ctx:    serverStream.ctx,
			cancel: serverStream.cancel,
		},
	}

	return pair
}

// LEARN: Flow control prevents overwhelming slow consumers
// Backpressure is essential for stable streaming.

// FlowControlledStream adds backpressure to a stream.
type FlowControlledStream[T any] struct {
	stream     Stream[T]
	maxPending int32
	pending    int32
	mu         sync.Mutex
	cond       *sync.Cond
}

// NewFlowControlledStream wraps a stream with flow control.
func NewFlowControlledStream[T any](stream Stream[T], maxPending int) *FlowControlledStream[T] {
	fcs := &FlowControlledStream[T]{
		stream:     stream,
		maxPending: int32(maxPending),
	}
	fcs.cond = sync.NewCond(&fcs.mu)
	return fcs
}

// Send blocks if too many messages are pending.
func (s *FlowControlledStream[T]) Send(msg T) error {
	s.mu.Lock()
	for atomic.LoadInt32(&s.pending) >= s.maxPending {
		// LEARN: Wait until there's room
		s.cond.Wait()

		// Check for cancellation
		select {
		case <-s.stream.Context().Done():
			s.mu.Unlock()
			return s.stream.Context().Err()
		default:
		}
	}
	atomic.AddInt32(&s.pending, 1)
	s.mu.Unlock()

	return s.stream.Send(msg)
}

// Recv receives and acknowledges receipt.
func (s *FlowControlledStream[T]) Recv() (T, error) {
	msg, err := s.stream.Recv()
	if err == nil {
		s.Ack()
	}
	return msg, err
}

// Ack acknowledges message processing, allowing more sends.
func (s *FlowControlledStream[T]) Ack() {
	s.mu.Lock()
	atomic.AddInt32(&s.pending, -1)
	s.cond.Signal()
	s.mu.Unlock()
}

// Close closes the underlying stream.
func (s *FlowControlledStream[T]) Close() error {
	return s.stream.Close()
}

// Context returns the stream's context.
func (s *FlowControlledStream[T]) Context() context.Context {
	return s.stream.Context()
}

// Pending returns the number of pending messages.
func (s *FlowControlledStream[T]) Pending() int {
	return int(atomic.LoadInt32(&s.pending))
}

// LEARN: Server streaming handler pattern
// Server sends multiple responses to single request.

// ServerStreamHandler handles server-side streaming.
type ServerStreamHandler[Req, Resp any] func(req Req, stream Stream[Resp]) error

// RunServerStream executes a server streaming handler.
func RunServerStream[Req, Resp any](
	ctx context.Context,
	req Req,
	handler ServerStreamHandler[Req, Resp],
	bufferSize int,
) (<-chan Resp, <-chan error) {
	respCh := make(chan Resp, bufferSize)
	errCh := make(chan error, 1)

	// Create stream for handler to write to
	stream := &sendOnlyStream[Resp]{
		ch:  respCh,
		ctx: ctx,
	}

	go func() {
		defer close(respCh)
		if err := handler(req, stream); err != nil {
			errCh <- err
		}
		close(errCh)
	}()

	return respCh, errCh
}

// sendOnlyStream is a stream that only supports Send.
type sendOnlyStream[T any] struct {
	ch     chan<- T
	ctx    context.Context
	closed int32
}

func (s *sendOnlyStream[T]) Send(msg T) error {
	if atomic.LoadInt32(&s.closed) == 1 {
		return ErrStreamClosed
	}

	select {
	case <-s.ctx.Done():
		return s.ctx.Err()
	case s.ch <- msg:
		return nil
	}
}

func (s *sendOnlyStream[T]) Recv() (T, error) {
	var zero T
	return zero, errors.New("receive not supported on send-only stream")
}

func (s *sendOnlyStream[T]) Close() error {
	atomic.StoreInt32(&s.closed, 1)
	return nil
}

func (s *sendOnlyStream[T]) Context() context.Context {
	return s.ctx
}

// LEARN: Client streaming handler pattern
// Client sends multiple requests for single response.

// ClientStreamHandler handles client-side streaming.
type ClientStreamHandler[Req, Resp any] func(stream Stream[Req]) (Resp, error)

// RunClientStream executes a client streaming handler.
func RunClientStream[Req, Resp any](
	ctx context.Context,
	handler ClientStreamHandler[Req, Resp],
	bufferSize int,
) (chan<- Req, <-chan Resp, <-chan error) {
	reqCh := make(chan Req, bufferSize)
	respCh := make(chan Resp, 1)
	errCh := make(chan error, 1)

	// Create stream for handler to read from
	stream := &recvOnlyStream[Req]{
		ch:  reqCh,
		ctx: ctx,
	}

	go func() {
		defer close(respCh)
		resp, err := handler(stream)
		if err != nil {
			errCh <- err
		} else {
			respCh <- resp
		}
		close(errCh)
	}()

	return reqCh, respCh, errCh
}

// recvOnlyStream is a stream that only supports Recv.
type recvOnlyStream[T any] struct {
	ch  <-chan T
	ctx context.Context
}

func (s *recvOnlyStream[T]) Send(_ T) error {
	return errors.New("send not supported on receive-only stream")
}

func (s *recvOnlyStream[T]) Recv() (T, error) {
	var zero T

	select {
	case <-s.ctx.Done():
		return zero, s.ctx.Err()
	case msg, ok := <-s.ch:
		if !ok {
			return zero, io.EOF
		}
		return msg, nil
	}
}

func (s *recvOnlyStream[T]) Close() error {
	return nil
}

func (s *recvOnlyStream[T]) Context() context.Context {
	return s.ctx
}

// LEARN: Bidirectional streaming
// Both sides send and receive concurrently.

// BidirectionalHandler handles bidirectional streaming.
type BidirectionalHandler[Req, Resp any] func(stream Stream[Req], respStream Stream[Resp]) error

// BidiStreamPair holds streams for bidirectional communication.
type BidiStreamPair[Req, Resp any] struct {
	ReqStream  *ChannelStream[Req]
	RespStream *ChannelStream[Resp]
}

// NewBidiStreamPair creates streams for bidirectional streaming.
func NewBidiStreamPair[Req, Resp any](ctx context.Context, bufferSize int) *BidiStreamPair[Req, Resp] {
	return &BidiStreamPair[Req, Resp]{
		ReqStream:  NewChannelStream[Req](ctx, bufferSize),
		RespStream: NewChannelStream[Resp](ctx, bufferSize),
	}
}

// LEARN: Streaming utilities

// CollectStream collects all messages from a stream into a slice.
func CollectStream[T any](stream Stream[T]) ([]T, error) {
	var result []T

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return result, err
		}
		result = append(result, msg)
	}
}

// SendAll sends all items from a slice to a stream.
func SendAll[T any](stream Stream[T], items []T) error {
	for _, item := range items {
		if err := stream.Send(item); err != nil {
			return err
		}
	}
	return nil
}

// LEARN: Timeout wrapper for streams

// TimeoutStream wraps a stream with per-operation timeouts.
type TimeoutStream[T any] struct {
	stream      Stream[T]
	sendTimeout time.Duration
	recvTimeout time.Duration
}

// NewTimeoutStream creates a stream with operation timeouts.
func NewTimeoutStream[T any](stream Stream[T], sendTimeout, recvTimeout time.Duration) *TimeoutStream[T] {
	return &TimeoutStream[T]{
		stream:      stream,
		sendTimeout: sendTimeout,
		recvTimeout: recvTimeout,
	}
}

// Send sends with timeout.
func (s *TimeoutStream[T]) Send(msg T) error {
	ctx, cancel := context.WithTimeout(s.stream.Context(), s.sendTimeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.stream.Send(msg)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Recv receives with timeout.
func (s *TimeoutStream[T]) Recv() (T, error) {
	var zero T

	ctx, cancel := context.WithTimeout(s.stream.Context(), s.recvTimeout)
	defer cancel()

	type result struct {
		msg T
		err error
	}

	done := make(chan result, 1)
	go func() {
		msg, err := s.stream.Recv()
		done <- result{msg, err}
	}()

	select {
	case <-ctx.Done():
		return zero, ctx.Err()
	case r := <-done:
		return r.msg, r.err
	}
}

// Close closes the underlying stream.
func (s *TimeoutStream[T]) Close() error {
	return s.stream.Close()
}

// Context returns the stream's context.
func (s *TimeoutStream[T]) Context() context.Context {
	return s.stream.Context()
}

// Stream errors
var (
	ErrStreamClosed = errors.New("stream closed")
)
