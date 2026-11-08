package main

import (
	"context"
	"database/sql"
	"net"
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

func listen(ctx context.Context) (net.Listener, error) {
	var lc net.ListenConfig
	return lc.Listen(ctx, "tcp", "127.0.0.1:0")
}

func listenPacket(ctx context.Context) (net.PacketConn, error) {
	var pc net.ListenConfig
	return pc.ListenPacket(ctx, "udp", "127.0.0.1:0")
}

func mirror(w http.ResponseWriter, path string) error {
	return store.genWrite(path)
}
