package store

import (
	"bufio"
	"io"
	"os"
	"syscall"
)

// MmapFile maps a file into memory: mmap is file I/O the gate must see even
// though no os.Create/Write pair is involved.
func MmapFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return syscall.Mmap(int(f.Fd()), 0, 4096, syscall.PROT_READ, syscall.MAP_PRIVATE)
}

// Buffered wraps a stream in a bounded reader: the buffered-write/bounded
// disciplines reason about exactly these constructors.
func Buffered(r io.Reader) *bufio.Reader {
	return bufio.NewReader(r)
}
