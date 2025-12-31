//go:build windows

// pkg/systems/mmap_windows.go
// Windows-specific mmap implementation using CreateFileMapping/MapViewOfFile
//
// LEARN: Windows uses a different API than Unix for memory mapping:
// - CreateFileMapping creates a file mapping object
// - MapViewOfFile maps a view of the file into memory
// - UnmapViewOfFile releases the mapping
// - CloseHandle closes the mapping object

package systems

import (
	"golang.org/x/sys/windows"
	"unsafe"
)

// windowsHandle stores Windows-specific resources
type windowsHandle struct {
	mapping windows.Handle
	addr    uintptr
}

// mmapPlatformOpen creates a Windows memory mapping.
func mmapPlatformOpen(m *mmapFile) error {
	// Determine protection flags
	var prot uint32 = windows.PAGE_READONLY
	var access uint32 = windows.FILE_MAP_READ
	if !m.readOnly {
		prot = windows.PAGE_READWRITE
		access = windows.FILE_MAP_WRITE
	}

	// Create file mapping object
	//
	// LEARN: CreateFileMapping creates a kernel object that represents
	// the mapping. Multiple processes can open the same mapping by name.
	// We use 0 for sizeLow/sizeHigh to map the entire file.
	mapping, err := windows.CreateFileMapping(
		windows.Handle(m.file.Fd()), // File handle
		nil,                         // Security attributes
		prot,                        // Page protection
		0,                           // Maximum size high
		0,                           // Maximum size low (0 = file size)
		nil,                         // Name (nil = anonymous)
	)
	if err != nil {
		return err
	}

	// Map view of file into memory
	//
	// LEARN: MapViewOfFile actually maps pages into our address space.
	// The returned pointer can be used like any memory address.
	addr, err := windows.MapViewOfFile(
		mapping, // Mapping handle
		access,  // Desired access
		0,       // Offset high
		0,       // Offset low
		0,       // Number of bytes (0 = entire file)
	)
	if err != nil {
		windows.CloseHandle(mapping)
		return err
	}

	// Create byte slice from mapped memory
	//
	// LEARN: This is the dangerous part - we're creating a Go slice
	// that points to OS-managed memory. The slice is only valid
	// while the mapping exists.
	m.data = unsafe.Slice((*byte)(unsafe.Pointer(addr)), int(m.size))

	// Store handles for cleanup
	m.platformData = &windowsHandle{
		mapping: mapping,
		addr:    addr,
	}

	return nil
}

// mmapPlatformClose releases Windows mapping resources.
func mmapPlatformClose(m *mmapFile) error {
	wh, ok := m.platformData.(*windowsHandle)
	if !ok || wh == nil {
		return nil
	}

	var firstErr error

	// Unmap the view
	//
	// LEARN: After UnmapViewOfFile, any access to m.data is undefined.
	// This is why we must ensure no references escape Close().
	if err := windows.UnmapViewOfFile(wh.addr); err != nil && firstErr == nil {
		firstErr = err
	}

	// Close the mapping handle
	if err := windows.CloseHandle(wh.mapping); err != nil && firstErr == nil {
		firstErr = err
	}

	// Clear data slice
	m.data = nil
	m.platformData = nil

	return firstErr
}

// mmapPlatformSync flushes Windows mapping to disk.
func mmapPlatformSync(m *mmapFile) error {
	if m.readOnly {
		return nil // Nothing to sync for read-only mappings
	}

	wh, ok := m.platformData.(*windowsHandle)
	if !ok || wh == nil {
		return ErrMmapClosed
	}

	// FlushViewOfFile writes modified pages to disk
	//
	// LEARN: Modified pages may be written to disk lazily by the OS.
	// Call FlushViewOfFile to force immediate write-back.
	return windows.FlushViewOfFile(wh.addr, uintptr(m.size))
}
