// Loop, timeout, and double-syscall fixture half (plan T12-T15).
package main

import (
	"net/http"
	"os"
)

func deferCloseInLoop(paths []string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		defer f.Close()
		_ = f
	}
}

func deferIOInLoop(dir string) {
	for i := 0; i < 3; i++ {
		defer os.MkdirAll(dir, 0o755)
	}
}

func deferOutsideLoop(path string) {
	defer os.Remove(path)
}

func closeInsideLoopNoDefer(paths []string) {
	for _, p := range paths {
		f, _ := os.Open(p)
		if f != nil {
			f.Close()
		}
	}
}

func statThenReadFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	return os.ReadFile(path)
}

func statLstatThenWrite(path string, data []byte) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func statDifferentPaths(a, b string) ([]byte, error) {
	if _, err := os.Stat(a); err != nil {
		return nil, err
	}

	return os.ReadFile(b)
}

func clientWithoutTimeout() *http.Client {
	return &http.Client{}
}

func clientWithTransport() *http.Client {
	return &http.Client{Transport: http.DefaultTransport}
}

func clientWithTimeout(timeout int) *http.Client {
	return &http.Client{Timeout: seconds(timeout)}
}

func defaultsInFunc(url string) error {
	resp, err := http.Post(url, "application/json", nil)
	_ = resp

	return err
}

func seconds(n int) timeDuration { return timeDuration(n) }

type timeDuration int

func atomicWriteDoneRight(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmp, path)
}

func seamAtomicWrite(path string, data []byte) error {
	return atomicWriteFile(path, data)
}

func atomicWriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0o644)
}

func tinyBufferReads(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 512)
	for {
		if _, err := f.Read(buf); err != nil {
			break
		}
	}

	return nil
}

func bigBufferReads(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 65536)
	for {
		if _, err := f.Read(buf); err != nil {
			break
		}
	}

	return nil
}

func tinyBufferScanner(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := newScanner(f)
	buf := make([]byte, 128)
	_ = buf

	return scanAll(scanner)
}

type scanner = int

func newScanner(f *os.File) scanner  { return 0 }
func scanAll(s scanner) error        { return nil }
