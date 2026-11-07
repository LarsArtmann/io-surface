// Command app is the detector fixture: deliberately bug-shaped code that is
// NEVER compiled (testdata trees are invisible to the go tool) — the
// extractor parses it, so every form must be parseable but correctness is
// not required. Each function exercises one err-dropped shape.
package main

import (
	"os"
)

func positiveBare(path string) {
	os.Remove(path)
}

func positiveBlank(path string) ([]byte, error) {
	data, _ := os.ReadFile(path)

	return data, nil
}

func positivePartialBlank(path string) {
	_, _ = os.Stat(path)
}

func negativeDeliberate(path string) {
	_ = os.Remove(path)
}

func negativeAllBlank(path string) {
	_, _ = os.Remove(path)
}

func negativeHandled(path string) error {
	_, err := os.Stat(path)

	return err
}

func negativePropagate(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func negativeNamed(path string) ([]byte, error) {
	data, err := os.ReadFile(path)

	return data, err
}

func negativeDeferred(dir string) {
	defer os.MkdirAll(dir, 0o755)
}

func negativeExpression(path string) {
	println(len(os.Stat(path)))
}
