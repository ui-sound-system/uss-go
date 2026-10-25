//go:build windows && amd64

package latency

import (
	"fmt"
	"unsafe"
)

var (
	getStdHandle     = kernel32.NewProc("GetStdHandle")
	getConsoleMode   = kernel32.NewProc("GetConsoleMode")
	setConsoleMode   = kernel32.NewProc("SetConsoleMode")
	readConsoleInput = kernel32.NewProc("ReadConsoleInputW")
)

// INPUT_RECORD's union begins at offset 4; KEY_EVENT_RECORD is 16 bytes.
type inputRecord struct {
	kind, pad                    uint16
	down                         int32
	repeat, key, scan, character uint16
	control                      uint32
}

var _ [20]byte = [unsafe.Sizeof(inputRecord{})]byte{}
var _ [4]byte = [unsafe.Offsetof(inputRecord{}.down)]byte{}

type console struct {
	handle   uintptr
	original uint32
	held     [256]bool
}

func openConsole() (*console, error) {
	handle, _, err := getStdHandle.Call(uintptr(^uint32(9))) // STD_INPUT_HANDLE = -10
	if handle == 0 || handle == ^uintptr(0) {
		return nil, fmt.Errorf("console input: %w", err)
	}
	c := &console{handle: handle}
	ok, _, err := getConsoleMode.Call(handle, uintptr(unsafe.Pointer(&c.original)))
	if ok == 0 {
		return nil, fmt.Errorf("use a Windows terminal, or --seconds 30 for automatic testing: %w", err)
	}
	mode := c.original & ^uint32(0x40|0x2|0x4) | 0x80 // Disable Quick Edit, line input and echo.
	ok, _, err = setConsoleMode.Call(handle, uintptr(mode))
	if ok == 0 {
		return nil, fmt.Errorf("console mode: %w", err)
	}
	return c, nil
}
func (c *console) close() { setConsoleMode.Call(c.handle, uintptr(c.original)) }
func (c *console) key() (uint16, error) {
	status, _, err := waitForSingle.Call(c.handle, 50)
	if status == 258 {
		return 0, nil
	}
	if status != 0 {
		return 0, fmt.Errorf("console wait: %w", err)
	}
	var record inputRecord
	var count uint32
	ok, _, err := readConsoleInput.Call(c.handle, uintptr(unsafe.Pointer(&record)), 1, uintptr(unsafe.Pointer(&count)))
	if ok == 0 {
		return 0, fmt.Errorf("console read: %w", err)
	}
	if count == 0 || record.kind != 1 || record.key >= 256 {
		return 0, nil
	}
	wasHeld := c.held[record.key]
	c.held[record.key] = record.down != 0
	if record.down == 0 || wasHeld {
		return 0, nil
	}
	return record.key, nil
}
