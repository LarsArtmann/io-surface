package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"

	"example.com/fixture/internal/store"
)

func fetch() (*http.Response, error) {
	return http.Get("https://example.test/x")
}

func openDB(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}

func serve(w http.ResponseWriter, r *http.Request, f *os.File) {
	http.ServeContent(w, r, "data.bin", time.Time{}, f)
}

func mirror(w http.ResponseWriter, path string) error {
	return store.genWrite(path)
}
