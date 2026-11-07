package store

import atomicwrite "github.com/larsartmann/go-atomic-write"

func SaveCheckpoint(path string, data []byte) error {
	return atomicwrite.WriteWithPerm(path, data, 0o644)
}
