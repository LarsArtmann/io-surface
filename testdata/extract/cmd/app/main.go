package main

import (
	"database/sql"
	"net/http"
)

func fetch() (*http.Response, error) {
	return http.Get("https://example.test/x")
}

func openDB(dsn string) (*sql.DB, error) {
	return sql.Open("sqlite", dsn)
}
