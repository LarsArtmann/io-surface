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
