// Package store is the closer-missing fixture half: every constructor,
// every close shape, and every ownership hand-off the rule must respect.
package store

import (
	"database/sql"
	"os"
)

func leakNeverUsed(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	_ = len(f.Name())

	return nil
}

func leakUsedNotClosed(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 32)
	n, _ := f.Read(buf)

	return buf[:n], nil
}

func positiveDeferred(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func positivePlainClose(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	if err := use(f); err != nil {
		f.Close()
	}

	return nil
}

func negativeReturned(path string) (*os.File, error) {
	return os.Open(path)
}

func negativeHandedOff(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	return use(f)
}

func negativeStored(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	sink = f

	return nil
}

func negativeBlank(path string) error {
	_, err := os.Create(path)

	return err
}

func negativeDBOpen(dsn string) (*sql.DB, error) {
	return sql.Open("postgres", dsn)
}

var sink *os.File

func use(f *os.File) error { return nil }
