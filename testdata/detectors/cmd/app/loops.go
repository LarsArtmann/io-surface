// Loop, timeout, and double-syscall fixture half (plan T12-T15).
package main

import (
	"bufio"
	"io"
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

func tinyBufferReadFull(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 512)
	for {
		if _, err := io.ReadFull(f, buf); err != nil {
			if errEOF(err) {
				break
			}

			return err
		}
	}

	return nil
}

func bigBufferReadFull(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 65536)
	for {
		if _, err := io.ReadFull(f, buf); err != nil {
			if errEOF(err) {
				break
			}

			return err
		}
	}

	return nil
}

func tinyReaderSize(path string) (*bufio.Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return bufio.NewReaderSize(f, 256), nil
}

func onceTinyBuffer(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	buf := make([]byte, 512)

	_, err = io.ReadFull(f, buf)

	return err
}

func errEOF(err error) bool { return err != nil && err.Error() == "EOF" }

func repeatTargetInLoop(path string) {
	for i := 0; i < 10; i++ {
		data, _ := os.ReadFile(path)
		_, _ = os.Stat(path)
		_ = len(data)
	}
}

func distinctTargetsInLoop(paths []string) {
	for _, p := range paths {
		data, _ := os.ReadFile(p)
		_ = len(data)
	}
}
