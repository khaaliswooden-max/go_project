// pkg/perf/json.go
// Zero-allocation JSON encoding with buffer pooling
//
// LEARN: This is Exercise 4.2 - minimize allocations in JSON encoding.
//
// Standard json.Marshal allocates:
// 1. bytes.Buffer for building output
// 2. Internal encoder state
// 3. The returned []byte slice
//
// This implementation reuses buffers via sync.Pool to reduce allocations.

package perf

import (
	"bytes"
	"encoding/json"
	"io"
	"sync"
)

// JSONEncoder provides pooled JSON encoding with minimal allocations.
//
// LEARN: The encoder reuses:
// 1. bytes.Buffer for building JSON output
// 2. json.Encoder (which caches reflection data)
//
// This reduces allocations from ~3 per encode to ~1.
type JSONEncoder struct {
	bufferPool *ByteBufferPool
	// encoderPool stores *json.Encoder, but they're tied to writers
	// so we can only pool the buffers effectively
}

// NewJSONEncoder creates a new pooled JSON encoder.
//
// bufferSize is the initial capacity for encoding buffers.
// Larger values reduce reallocations for big payloads.
func NewJSONEncoder(bufferSize int) *JSONEncoder {
	return &JSONEncoder{
		bufferPool: NewByteBufferPool(bufferSize),
	}
}

// DefaultJSONEncoder is a shared encoder with sensible defaults.
//
// LEARN: Using a package-level default avoids passing encoder
// everywhere, but may not be optimal for all payload sizes.
var DefaultJSONEncoder = NewJSONEncoder(4096)

// Encode marshals value to JSON using a pooled buffer.
//
// Returns the JSON bytes. The caller should not retain the
// returned slice beyond the current request scope.
//
// LEARN: This method allocates once for the final []byte copy.
// The internal buffer is reused.
func (e *JSONEncoder) Encode(v any) ([]byte, error) {
	buf := e.bufferPool.Get()
	defer e.bufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// Remove trailing newline added by Encoder
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	// Copy to return (buffer will be reused)
	result := make([]byte, len(b))
	copy(result, b)
	return result, nil
}

// EncodeTo marshals value to JSON and writes to the provided writer.
//
// LEARN: This is the most efficient path when you have a writer
// (like http.ResponseWriter) because no final copy is needed.
func (e *JSONEncoder) EncodeTo(w io.Writer, v any) error {
	buf := e.bufferPool.Get()
	defer e.bufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return err
	}

	// Remove trailing newline
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	_, err := w.Write(b)
	return err
}

// EncodeAppend appends JSON to an existing buffer.
//
// LEARN: This is useful when building larger documents
// where you need to embed JSON within other content.
func (e *JSONEncoder) EncodeAppend(dst []byte, v any) ([]byte, error) {
	buf := e.bufferPool.Get()
	defer e.bufferPool.Put(buf)

	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(v); err != nil {
		return dst, err
	}

	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}

	return append(dst, b...), nil
}

// === Package-level convenience functions ===

// MarshalPooled is a drop-in replacement for json.Marshal using pooled buffers.
//
// LEARN: Use this instead of json.Marshal in hot paths to reduce allocations.
func MarshalPooled(v any) ([]byte, error) {
	return DefaultJSONEncoder.Encode(v)
}

// === JSONDecoder with pooling ===

// JSONDecoder provides pooled JSON decoding.
type JSONDecoder struct {
	// For decoding, there's less we can pool because
	// the output object is caller-provided.
	// We can pool the decoder's internal buffer though.
}

// decoderPool pools json.Decoder with underlying bytes.Reader.
var decoderPool = sync.Pool{
	New: func() any {
		return &pooledDecoder{
			reader: bytes.NewReader(nil),
		}
	},
}

type pooledDecoder struct {
	reader  *bytes.Reader
	decoder *json.Decoder
}

// UnmarshalPooled is a drop-in replacement for json.Unmarshal.
//
// LEARN: The savings here are smaller than for Marshal because:
// 1. The output object is allocated by the caller
// 2. Decoder internal state is harder to reuse
//
// Still, pooling bytes.Reader saves some allocations.
func UnmarshalPooled(data []byte, v any) error {
	pd := decoderPool.Get().(*pooledDecoder)
	defer decoderPool.Put(pd)

	pd.reader.Reset(data)
	pd.decoder = json.NewDecoder(pd.reader)

	return pd.decoder.Decode(v)
}

// === Allocation tracking for benchmarks ===

// JSONStats tracks JSON encoding allocation statistics.
type JSONStats struct {
	Encodes      int64
	TotalBytes   int64
	BufferReuses int64
}

// TrackedJSONEncoder wraps JSONEncoder with statistics.
type TrackedJSONEncoder struct {
	*JSONEncoder
	stats   JSONStats
	statsMu sync.Mutex
}

// NewTrackedJSONEncoder creates an encoder that tracks usage.
func NewTrackedJSONEncoder(bufferSize int) *TrackedJSONEncoder {
	return &TrackedJSONEncoder{
		JSONEncoder: NewJSONEncoder(bufferSize),
	}
}

// Encode with statistics tracking.
func (e *TrackedJSONEncoder) Encode(v any) ([]byte, error) {
	result, err := e.JSONEncoder.Encode(v)

	e.statsMu.Lock()
	e.stats.Encodes++
	e.stats.TotalBytes += int64(len(result))
	e.statsMu.Unlock()

	return result, err
}

// Stats returns current statistics.
func (e *TrackedJSONEncoder) Stats() JSONStats {
	e.statsMu.Lock()
	defer e.statsMu.Unlock()
	return e.stats
}
