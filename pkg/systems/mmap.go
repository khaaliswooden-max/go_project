// pkg/systems/mmap.go
// Memory-mapped file abstraction for efficient large file access
//
// LEARN: Memory mapping treats files as if they were arrays in memory.
// The OS handles paging data in/out, enabling efficient random access
// to files larger than RAM.
//
// Key concepts:
// 1. Lazy loading: Data loads on-demand via page faults
// 2. Shared caching: Multiple processes can share mapped pages
// 3. Platform-specific: Windows and Unix have different APIs

package systems

import (
	"errors"
	"io"
	"os"
	"sync"
)

// Common errors for mmap operations
var (
	ErrMmapClosed     = errors.New("mmap: file is closed")
	ErrMmapEmpty      = errors.New("mmap: cannot map empty file")
	ErrMmapTooLarge   = errors.New("mmap: file too large to map")
	ErrMmapWriteRO    = errors.New("mmap: cannot write to read-only mapping")
	ErrMmapOutOfRange = errors.New("mmap: access out of range")
)

// MappedFile represents a memory-mapped file.
//
// LEARN: This interface abstracts platform differences between
// Windows (CreateFileMapping/MapViewOfFile) and Unix (mmap syscall).
type MappedFile interface {
	// Data returns the mapped memory region as a byte slice.
	// The slice is valid only until Close() is called.
	Data() []byte
	
	// Len returns the length of the mapping.
	Len() int
	
	// ReadAt reads len(p) bytes from offset into p.
	// Implements io.ReaderAt for compatibility.
	ReadAt(p []byte, off int64) (n int, err error)
	
	// Close unmaps the file and releases resources.
	// After Close, Data() must not be accessed.
	Close() error
	
	// Sync flushes changes to disk (for writable mappings).
	Sync() error
}

// MmapOptions configures memory mapping behavior.
type MmapOptions struct {
	// ReadOnly maps the file for reading only.
	// Attempting to write will cause an error or segfault.
	ReadOnly bool
	
	// MaxSize limits the mapping size (0 = no limit).
	// Useful for preventing excessive virtual address consumption.
	MaxSize int64
}

// DefaultMmapOptions returns sensible defaults for mmap.
func DefaultMmapOptions() MmapOptions {
	return MmapOptions{
		ReadOnly: true,
		MaxSize:  1 << 30, // 1GB limit
	}
}

// === Portable Implementation ===

// mmapFile is the core implementation of MappedFile.
// Platform-specific details are handled by mmapImpl.
type mmapFile struct {
	data     []byte    // Mapped memory region
	file     *os.File  // Underlying file handle
	size     int64     // Mapping size
	readOnly bool      // Read-only flag
	closed   bool      // Close state
	mu       sync.RWMutex // Protects closed state
	
	// Platform-specific resources (set by mmapPlatformOpen)
	platformData any
}

// OpenMmap opens a file for memory-mapped reading.
//
// LEARN: This is the main entry point for mmap. It:
// 1. Opens and stats the file
// 2. Validates size constraints
// 3. Calls platform-specific mapping code
//
// Example:
//
//	mf, err := OpenMmap("large_file.bin", DefaultMmapOptions())
//	if err != nil {
//	    return err
//	}
//	defer mf.Close()
//	
//	data := mf.Data()
//	// Access data directly - OS handles paging
//	fmt.Println(data[0:100])
func OpenMmap(path string, opts MmapOptions) (MappedFile, error) {
	// Open file
	flags := os.O_RDONLY
	if !opts.ReadOnly {
		flags = os.O_RDWR
	}
	
	f, err := os.OpenFile(path, flags, 0)
	if err != nil {
		return nil, err
	}
	
	// Get file size
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	
	size := info.Size()
	if size == 0 {
		f.Close()
		return nil, ErrMmapEmpty
	}
	
	if opts.MaxSize > 0 && size > opts.MaxSize {
		f.Close()
		return nil, ErrMmapTooLarge
	}
	
	// Create mapping
	mf := &mmapFile{
		file:     f,
		size:     size,
		readOnly: opts.ReadOnly,
	}
	
	// Platform-specific mapping
	if err := mmapPlatformOpen(mf); err != nil {
		f.Close()
		return nil, err
	}
	
	return mf, nil
}

// Data returns the mapped memory region.
func (m *mmapFile) Data() []byte {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return nil
	}
	return m.data
}

// Len returns the length of the mapping.
func (m *mmapFile) Len() int {
	return int(m.size)
}

// ReadAt implements io.ReaderAt.
func (m *mmapFile) ReadAt(p []byte, off int64) (n int, err error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return 0, ErrMmapClosed
	}
	
	if off < 0 || off >= m.size {
		return 0, ErrMmapOutOfRange
	}
	
	n = copy(p, m.data[off:])
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Close unmaps the file and releases resources.
func (m *mmapFile) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.closed {
		return nil
	}
	m.closed = true
	
	// Platform-specific cleanup
	err := mmapPlatformClose(m)
	
	// Close file
	if ferr := m.file.Close(); err == nil {
		err = ferr
	}
	
	return err
}

// Sync flushes changes to disk.
func (m *mmapFile) Sync() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.closed {
		return ErrMmapClosed
	}
	
	return mmapPlatformSync(m)
}

// === Memory-Mapped Reader ===

// MmapReader wraps a MappedFile as an io.Reader for streaming access.
//
// LEARN: Even with mmap, sometimes you need streaming semantics.
// This wrapper provides io.Reader compatibility while using mmap underneath.
type MmapReader struct {
	mf     MappedFile
	offset int64
}

// NewMmapReader creates a streaming reader from a mapped file.
func NewMmapReader(mf MappedFile) *MmapReader {
	return &MmapReader{mf: mf, offset: 0}
}

// Read implements io.Reader.
func (r *MmapReader) Read(p []byte) (n int, err error) {
	n, err = r.mf.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// Seek implements io.Seeker.
func (r *MmapReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.offset + offset
	case io.SeekEnd:
		abs = int64(r.mf.Len()) + offset
	default:
		return 0, errors.New("mmap: invalid whence")
	}
	
	if abs < 0 {
		return 0, errors.New("mmap: negative position")
	}
	
	r.offset = abs
	return abs, nil
}

// === Utility Functions ===

// ReadFileMmap reads an entire file using mmap.
//
// LEARN: For one-shot reads, mmap may not be faster than os.ReadFile
// due to mapping overhead. Use mmap for:
// - Random access to large files
// - Repeated access patterns
// - Sharing data between processes
func ReadFileMmap(path string) ([]byte, error) {
	mf, err := OpenMmap(path, DefaultMmapOptions())
	if err != nil {
		// Fallback to regular read
		return os.ReadFile(path)
	}
	defer mf.Close()
	
	// Copy data since mmap data is invalid after Close
	data := mf.Data()
	result := make([]byte, len(data))
	copy(result, data)
	
	return result, nil
}

// SearchMmap searches for a pattern in a memory-mapped file.
//
// LEARN: This demonstrates mmap's random access benefits.
// The OS pages in data on-demand as we scan.
func SearchMmap(mf MappedFile, pattern []byte) int64 {
	data := mf.Data()
	if len(pattern) == 0 || len(data) < len(pattern) {
		return -1
	}
	
	// Simple search - could use Boyer-Moore for large patterns
	for i := 0; i <= len(data)-len(pattern); i++ {
		match := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				match = false
				break
			}
		}
		if match {
			return int64(i)
		}
	}
	
	return -1
}

