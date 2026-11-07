// Context-drop fixture half (plan T8/T9): variant pairs fire only when a
// context.Context is in scope; the no-ctx and receiver-ctx negatives stay
// silent.
package main

import (
	"context"
	"net"
	"net/http"
	"os/exec"
)

func positiveExec(ctx context.Context) error {
	cmd := exec.Command("git", "status")

	return cmd.Run()
}

func positiveDial(ctx context.Context) error {
	conn, err := net.Dial("tcp", "host.test:80")
	_ = conn

	return err
}

func positiveDialTimeout(ctx context.Context) error {
	conn, err := net.DialTimeout("tcp", "host.test:80", 5*1e9)
	_ = conn

	return err
}

func positiveGet(ctx context.Context) error {
	resp, err := http.Get("https://host.test/x")
	_ = resp

	return err
}

func negativeNoCtx() error {
	conn, err := net.Dial("tcp", "host.test:80")
	_ = conn

	return err
}

func negativeReceiverCtx(s *session) error {
	conn, err := net.Dial("tcp", s.host)
	_ = conn

	return err
}

func negativeCtxVariant(ctx context.Context) error {
	d := net.Dialer{}
	conn, err := d.DialContext(ctx, "tcp", "host.test:80")
	_ = conn

	return err
}

type session struct{ host string }
