// pkg/systems/mmap_test.go
// Tests for memory-mapped file operations

package systems

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// createTestFile creates a temporary file with the given content
func createTestFile(t *testing.T, content []byte) string {
	t.Helper()
	
	f, err := os.CreateTemp("", "mmap_test_*")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		t.Fatalf("failed to write temp file: %v", err)
	}
	
	if err := f.Close(); err != nil {
		os.Remove(f.Name())
		t.Fatalf("failed to close temp file: %v", err)
	}
	
	t.Cleanup(func() {
		os.Remove(f.Name())
	})
	
	return f.Name()
}

// === OpenMmap Tests ===

func TestOpenMmap_Basic(t *testing.T) {
	content := []byte("Hello, mmap!")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	// Verify data matches
	data := mf.Data()
	if !bytes.Equal(data, content) {
		t.Errorf("Data mismatch: got %q, want %q", data, content)
	}
	
	// Verify length
	if mf.Len() != len(content) {
		t.Errorf("Len = %d, want %d", mf.Len(), len(content))
	}
}

func TestOpenMmap_LargeFile(t *testing.T) {
	// Create a 1MB file
	size := 1 << 20
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}
	
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	// Verify data
	data := mf.Data()
	if len(data) != size {
		t.Errorf("Size = %d, want %d", len(data), size)
	}
	
	// Check some random positions
	positions := []int{0, 100, 1000, 10000, 100000, size - 1}
	for _, pos := range positions {
		if data[pos] != content[pos] {
			t.Errorf("data[%d] = %d, want %d", pos, data[pos], content[pos])
		}
	}
}

func TestOpenMmap_EmptyFile(t *testing.T) {
	path := createTestFile(t, []byte{})
	
	_, err := OpenMmap(path, DefaultMmapOptions())
	if err != ErrMmapEmpty {
		t.Errorf("Expected ErrMmapEmpty, got %v", err)
	}
}

func TestOpenMmap_NonexistentFile(t *testing.T) {
	_, err := OpenMmap("/nonexistent/file/path", DefaultMmapOptions())
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestOpenMmap_TooLarge(t *testing.T) {
	content := make([]byte, 1000)
	path := createTestFile(t, content)
	
	opts := MmapOptions{
		ReadOnly: true,
		MaxSize:  100, // File is larger than this
	}
	
	_, err := OpenMmap(path, opts)
	if err != ErrMmapTooLarge {
		t.Errorf("Expected ErrMmapTooLarge, got %v", err)
	}
}

// === ReadAt Tests ===

func TestMmapFile_ReadAt(t *testing.T) {
	content := []byte("0123456789ABCDEF")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	tests := []struct {
		name   string
		buf    []byte
		offset int64
		want   []byte
		wantN  int
		wantErr error
	}{
		{"start", make([]byte, 4), 0, []byte("0123"), 4, nil},
		{"middle", make([]byte, 4), 4, []byte("4567"), 4, nil},
		{"end", make([]byte, 4), 12, []byte("CDEF"), 4, nil},
		{"partial", make([]byte, 10), 12, []byte("CDEF"), 4, io.EOF},
		{"out of range", make([]byte, 4), 20, nil, 0, ErrMmapOutOfRange},
		{"negative offset", make([]byte, 4), -1, nil, 0, ErrMmapOutOfRange},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			n, err := mf.ReadAt(tc.buf, tc.offset)
			
			if err != tc.wantErr {
				t.Errorf("err = %v, want %v", err, tc.wantErr)
			}
			
			if n != tc.wantN {
				t.Errorf("n = %d, want %d", n, tc.wantN)
			}
			
			if tc.want != nil && !bytes.Equal(tc.buf[:n], tc.want) {
				t.Errorf("data = %q, want %q", tc.buf[:n], tc.want)
			}
		})
	}
}

// === Close Tests ===

func TestMmapFile_Close(t *testing.T) {
	content := []byte("test data")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	
	// Verify data before close
	if mf.Data() == nil {
		t.Error("Data should not be nil before close")
	}
	
	// Close
	if err := mf.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
	
	// Verify data after close
	if mf.Data() != nil {
		t.Error("Data should be nil after close")
	}
	
	// Double close should be safe
	if err := mf.Close(); err != nil {
		t.Errorf("Double close failed: %v", err)
	}
}

func TestMmapFile_ReadAtAfterClose(t *testing.T) {
	content := []byte("test data")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	
	mf.Close()
	
	buf := make([]byte, 4)
	_, err = mf.ReadAt(buf, 0)
	if err != ErrMmapClosed {
		t.Errorf("Expected ErrMmapClosed, got %v", err)
	}
}

// === MmapReader Tests ===

func TestMmapReader(t *testing.T) {
	content := []byte("Hello, World! This is a test.")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	reader := NewMmapReader(mf)
	
	// Read in chunks
	var result []byte
	buf := make([]byte, 5)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			result = append(result, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read error: %v", err)
		}
	}
	
	if !bytes.Equal(result, content) {
		t.Errorf("Result = %q, want %q", result, content)
	}
}

func TestMmapReader_Seek(t *testing.T) {
	content := []byte("0123456789")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	reader := NewMmapReader(mf)
	
	tests := []struct {
		name    string
		offset  int64
		whence  int
		wantPos int64
		wantData string
	}{
		{"start", 3, io.SeekStart, 3, "345"},
		{"current+2", 2, io.SeekCurrent, 8, "89"},
		{"end-3", -3, io.SeekEnd, 7, "789"},
	}
	
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pos, err := reader.Seek(tc.offset, tc.whence)
			if err != nil {
				t.Fatalf("Seek error: %v", err)
			}
			if pos != tc.wantPos {
				t.Errorf("Position = %d, want %d", pos, tc.wantPos)
			}
			
			buf := make([]byte, len(tc.wantData))
			n, _ := reader.Read(buf)
			if string(buf[:n]) != tc.wantData {
				t.Errorf("Data = %q, want %q", buf[:n], tc.wantData)
			}
		})
	}
}

// === SearchMmap Tests ===

func TestSearchMmap(t *testing.T) {
	content := []byte("The quick brown fox jumps over the lazy dog")
	path := createTestFile(t, content)
	
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		t.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	tests := []struct {
		pattern string
		want    int64
	}{
		{"quick", 4},
		{"fox", 16},
		{"dog", 40},
		{"The", 0},
		{"cat", -1},
		{"", -1},
	}
	
	for _, tc := range tests {
		t.Run(tc.pattern, func(t *testing.T) {
			got := SearchMmap(mf, []byte(tc.pattern))
			if got != tc.want {
				t.Errorf("SearchMmap(%q) = %d, want %d", tc.pattern, got, tc.want)
			}
		})
	}
}

// === ReadFileMmap Tests ===

func TestReadFileMmap(t *testing.T) {
	content := []byte("File content for mmap read test")
	path := createTestFile(t, content)
	
	data, err := ReadFileMmap(path)
	if err != nil {
		t.Fatalf("ReadFileMmap failed: %v", err)
	}
	
	if !bytes.Equal(data, content) {
		t.Errorf("Data = %q, want %q", data, content)
	}
}

func TestReadFileMmap_Fallback(t *testing.T) {
	// Test with nonexistent file - should fail (no fallback for missing files)
	_, err := ReadFileMmap(filepath.Join(os.TempDir(), "nonexistent_file_12345"))
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

// === Benchmarks ===

func BenchmarkMmapVsReadFile(b *testing.B) {
	// Create a 1MB test file
	size := 1 << 20
	content := make([]byte, size)
	for i := range content {
		content[i] = byte(i % 256)
	}
	
	f, err := os.CreateTemp("", "mmap_bench_*")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	f.Write(content)
	f.Close()
	defer os.Remove(f.Name())
	
	b.Run("os.ReadFile", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			data, err := os.ReadFile(f.Name())
			if err != nil {
				b.Fatal(err)
			}
			_ = data[len(data)/2] // Access middle
		}
	})
	
	b.Run("Mmap", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			mf, err := OpenMmap(f.Name(), DefaultMmapOptions())
			if err != nil {
				b.Fatal(err)
			}
			data := mf.Data()
			_ = data[len(data)/2] // Access middle
			mf.Close()
		}
	})
	
	// Mmap with reuse (more realistic for random access)
	b.Run("Mmap_Reuse", func(b *testing.B) {
		mf, err := OpenMmap(f.Name(), DefaultMmapOptions())
		if err != nil {
			b.Fatal(err)
		}
		defer mf.Close()
		
		b.ReportAllocs()
		b.ResetTimer()
		
		data := mf.Data()
		for i := 0; i < b.N; i++ {
			// Random access pattern
			offset := (i * 1234) % len(data)
			_ = data[offset]
		}
	})
}

func BenchmarkSearchMmap(b *testing.B) {
	// Create a larger file for search benchmarks
	size := 1 << 20 // 1MB
	content := bytes.Repeat([]byte("The quick brown fox jumps over the lazy dog. "), size/46)
	
	f, err := os.CreateTemp("", "mmap_search_bench_*")
	if err != nil {
		b.Fatalf("failed to create temp file: %v", err)
	}
	f.Write(content)
	f.Close()
	defer os.Remove(f.Name())
	
	mf, err := OpenMmap(f.Name(), DefaultMmapOptions())
	if err != nil {
		b.Fatalf("OpenMmap failed: %v", err)
	}
	defer mf.Close()
	
	patterns := []string{"quick", "lazy dog", "nonexistent pattern here"}
	
	for _, pattern := range patterns {
		b.Run(pattern, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				SearchMmap(mf, []byte(pattern))
			}
		})
	}
}

