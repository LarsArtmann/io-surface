package store

import "os"

//go:embed quota.json
var quotaJSON string

func LoadConfig(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func WriteTemp(dir string) (*os.File, error) {
	return os.CreateTemp(dir, "cfg")
}

func persistAll(path string, data []byte) error {
	return atomicWriteFile(path, data)
}

func (s *Server) Serve() error {
	return s.ListenAndServe()
}

type Server struct{}

func atomicWriteFile(string, []byte) error { return nil }
