//go:build unix || darwin || linux || freebsd

// pkg/systems/mmap_unix.go
// Unix-specific mmap implementation using mmap(2) syscall
//
// LEARN: Unix mmap is simpler than Windows:
// - mmap() maps file into memory
// - munmap() releases the mapping
// - msync() flushes changes to disk

package systems

import (
	"golang.org/x/sys/unix"
)

// mmapPlatformOpen creates a Unix memory mapping.
func mmapPlatformOpen(m *mmapFile) error {
	// Determine protection flags
	//
	// LEARN: mmap protection flags:
	// - PROT_READ: Pages can be read
	// - PROT_WRITE: Pages can be written
	// - PROT_EXEC: Pages can be executed (not used for files)
	prot := unix.PROT_READ
	if !m.readOnly {
		prot |= unix.PROT_WRITE
	}
	
	// Map the file
	//
	// LEARN: mmap flags:
	// - MAP_SHARED: Changes are shared with other processes and written to file
	// - MAP_PRIVATE: Changes are copy-on-write, not written to file
	// We use MAP_SHARED for writable mappings to enable Sync().
	flags := unix.MAP_SHARED
	
	data, err := unix.Mmap(
		int(m.file.Fd()), // File descriptor
		0,                 // Offset (map from beginning)
		int(m.size),       // Length
		prot,              // Protection flags
		flags,             // Mapping flags
	)
	if err != nil {
		return err
	}
	
	m.data = data
	// Unix mmap doesn't need additional handles; data slice IS the resource
	m.platformData = nil
	
	return nil
}

// mmapPlatformClose releases Unix mapping resources.
func mmapPlatformClose(m *mmapFile) error {
	if m.data == nil {
		return nil
	}
	
	// munmap releases the mapping
	//
	// LEARN: After munmap, accessing m.data is a segfault.
	// Go's GC doesn't know about mmap, so we must manually clean up.
	err := unix.Munmap(m.data)
	m.data = nil
	
	return err
}

// mmapPlatformSync flushes Unix mapping to disk.
func mmapPlatformSync(m *mmapFile) error {
	if m.readOnly || m.data == nil {
		return nil
	}
	
	// msync forces write-back of modified pages
	//
	// LEARN: msync flags:
	// - MS_SYNC: Synchronous - wait for write to complete
	// - MS_ASYNC: Asynchronous - schedule write but don't wait
	// - MS_INVALIDATE: Invalidate other mappings (rarely used)
	return unix.Msync(m.data, unix.MS_SYNC)
}

