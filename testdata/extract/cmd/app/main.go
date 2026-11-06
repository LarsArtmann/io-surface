package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"
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
