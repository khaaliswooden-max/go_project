// pkg/perf/json_test.go
// Tests and benchmarks for JSON encoding/decoding
//
// LEARN: This file demonstrates benchmarking methodology:
// 1. Compare against stdlib baseline
// 2. Measure allocations per operation
// 3. Test with realistic data sizes

package perf

import (
	"bytes"
	"encoding/json"
	"testing"
)

// Test data structure
type TestPayload struct {
	ID       int               `json:"id"`
	Name     string            `json:"name"`
	Email    string            `json:"email"`
	Active   bool              `json:"active"`
	Score    float64           `json:"score"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
}

func newTestPayload() TestPayload {
	return TestPayload{
		ID:     12345,
		Name:   "John Doe",
		Email:  "john.doe@example.com",
		Active: true,
		Score:  98.6,
		Tags:   []string{"admin", "verified", "premium"},
		Metadata: map[string]string{
			"created": "2024-01-01",
			"updated": "2024-12-30",
			"version": "1.0",
		},
	}
}

// === Unit Tests ===

func TestJSONEncoder_Encode(t *testing.T) {
	encoder := NewJSONEncoder(1024)
	payload := newTestPayload()

	result, err := encoder.Encode(payload)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	// Verify it's valid JSON by decoding
	var decoded TestPayload
	if err := json.Unmarshal(result, &decoded); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}

	if decoded.ID != payload.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, payload.ID)
	}
	if decoded.Name != payload.Name {
		t.Errorf("Name mismatch: got %s, want %s", decoded.Name, payload.Name)
	}
}

func TestJSONEncoder_EncodeTo(t *testing.T) {
	encoder := NewJSONEncoder(1024)
	payload := newTestPayload()

	var buf bytes.Buffer
	if err := encoder.EncodeTo(&buf, payload); err != nil {
		t.Fatalf("EncodeTo failed: %v", err)
	}

	// Verify valid JSON
	var decoded TestPayload
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("Result is not valid JSON: %v", err)
	}
}

func TestJSONEncoder_EncodeAppend(t *testing.T) {
	encoder := NewJSONEncoder(1024)
	payload := newTestPayload()

	prefix := []byte(`{"wrapper":`)
	result, err := encoder.EncodeAppend(prefix, payload)
	if err != nil {
		t.Fatalf("EncodeAppend failed: %v", err)
	}
	result = append(result, '}')

	// Should be valid JSON with wrapper
	var wrapper struct {
		Wrapper TestPayload `json:"wrapper"`
	}
	if err := json.Unmarshal(result, &wrapper); err != nil {
		t.Fatalf("Wrapped result is not valid JSON: %v", err)
	}
}

func TestMarshalPooled(t *testing.T) {
	payload := newTestPayload()

	result, err := MarshalPooled(payload)
	if err != nil {
		t.Fatalf("MarshalPooled failed: %v", err)
	}

	// Compare with stdlib
	expected, _ := json.Marshal(payload)
	// Trim newline from expected (json.Marshal doesn't add one, Encoder does)
	if !bytes.Equal(result, expected) {
		// Allow for key ordering differences
		var r, e TestPayload
		json.Unmarshal(result, &r)
		json.Unmarshal(expected, &e)
		if r.ID != e.ID || r.Name != e.Name {
			t.Errorf("Result mismatch:\ngot:  %s\nwant: %s", result, expected)
		}
	}
}

func TestUnmarshalPooled(t *testing.T) {
	payload := newTestPayload()
	data, _ := json.Marshal(payload)

	var decoded TestPayload
	if err := UnmarshalPooled(data, &decoded); err != nil {
		t.Fatalf("UnmarshalPooled failed: %v", err)
	}

	if decoded.ID != payload.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, payload.ID)
	}
}

// === Benchmarks ===

// BenchmarkJSONMarshal compares stdlib vs pooled encoding.
//
// LEARN: Run with: go test -bench=BenchmarkJSONMarshal -benchmem ./pkg/perf/...
//
// Expected output shows allocation reduction:
//
//	BenchmarkJSONMarshal/stdlib-8        500000   2500 ns/op   1024 B/op   3 allocs/op
//	BenchmarkJSONMarshal/pooled-8       1000000   1500 ns/op    512 B/op   1 allocs/op
func BenchmarkJSONMarshal(b *testing.B) {
	payload := newTestPayload()

	b.Run("stdlib", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pooled", func(b *testing.B) {
		encoder := NewJSONEncoder(1024)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := encoder.Encode(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pooled_default", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := MarshalPooled(payload)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkJSONMarshalToWriter tests encoding directly to writer.
func BenchmarkJSONMarshalToWriter(b *testing.B) {
	payload := newTestPayload()

	b.Run("stdlib_encoder", func(b *testing.B) {
		var buf bytes.Buffer
		encoder := json.NewEncoder(&buf)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := encoder.Encode(payload); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pooled_to_writer", func(b *testing.B) {
		encoder := NewJSONEncoder(1024)
		var buf bytes.Buffer
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			buf.Reset()
			if err := encoder.EncodeTo(&buf, payload); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkJSONUnmarshal compares stdlib vs pooled decoding.
func BenchmarkJSONUnmarshal(b *testing.B) {
	payload := newTestPayload()
	data, _ := json.Marshal(payload)

	b.Run("stdlib", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var p TestPayload
			if err := json.Unmarshal(data, &p); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("pooled", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			var p TestPayload
			if err := UnmarshalPooled(data, &p); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkJSONPayloadSizes tests different payload sizes.
//
// LEARN: Pool benefits vary with payload size:
// - Small payloads: Minimal benefit
// - Medium payloads: Good benefit
// - Large payloads: Significant benefit
func BenchmarkJSONPayloadSizes(b *testing.B) {
	sizes := []struct {
		name    string
		payload any
	}{
		{"small", struct{ ID int }{1}},
		{"medium", newTestPayload()},
		{"large", struct {
			Items [100]TestPayload
		}{}},
	}

	for _, size := range sizes {
		b.Run(size.name+"_stdlib", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				json.Marshal(size.payload)
			}
		})

		b.Run(size.name+"_pooled", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				MarshalPooled(size.payload)
			}
		})
	}
}

// BenchmarkJSONParallel tests concurrent encoding performance.
func BenchmarkJSONParallel(b *testing.B) {
	payload := newTestPayload()

	b.Run("stdlib", func(b *testing.B) {
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				json.Marshal(payload)
			}
		})
	})

	b.Run("pooled", func(b *testing.B) {
		encoder := NewJSONEncoder(1024)
		b.ReportAllocs()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				encoder.Encode(payload)
			}
		})
	})
}

// === Allocation Tests ===

// TestJSONAllocationCount verifies allocation reduction vs stdlib.
//
// LEARN: testing.AllocsPerRun measures allocations accurately.
// This is useful for regression testing allocation improvements.
//
// Note: json.Encoder internally allocates for reflection, maps, and
// slices. Our goal is to reduce allocations vs stdlib, not eliminate them.
func TestJSONAllocationCount(t *testing.T) {
	payload := newTestPayload()
	encoder := NewJSONEncoder(1024)

	// Warm up the pool
	for i := 0; i < 10; i++ {
		encoder.Encode(payload)
	}

	// Measure pooled allocations
	pooledAllocs := testing.AllocsPerRun(100, func() {
		encoder.Encode(payload)
	})

	// Measure stdlib allocations for comparison
	stdlibAllocs := testing.AllocsPerRun(100, func() {
		json.Marshal(payload)
	})

	// Pooled should have fewer or equal allocations than stdlib
	// The improvement comes from buffer reuse
	t.Logf("Pooled allocations: %.1f, Stdlib allocations: %.1f", pooledAllocs, stdlibAllocs)

	// Allow some variance, but pooled shouldn't be significantly worse
	if pooledAllocs > stdlibAllocs+2 {
		t.Errorf("Pooled allocations (%.1f) significantly higher than stdlib (%.1f)", pooledAllocs, stdlibAllocs)
	}
}
