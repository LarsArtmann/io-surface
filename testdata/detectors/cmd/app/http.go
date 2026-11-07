// HTTP body-close fixture half (plan T7): every resp.Body shape the
// closer-missing variant must classify.
package main

import (
	"io"
	"net/http"
)

func leakBody(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	_, _ = io.ReadAll(resp.Body)

	return nil
}

func statusOnlyLeak(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return errStatus
	}

	return nil
}

func closedBody(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func plainCloseBody(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	_, _ = io.ReadAll(resp.Body)
	resp.Body.Close()

	return nil
}

func transferredResp(url string) (*http.Response, error) {
	return http.Get(url)
}

func handedOffResp(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	return drain(resp)
}

func errSentinel() error { return errStatus }

func drain(r *http.Response) error { return nil }

var errStatus = error(nil)
