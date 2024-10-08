// Package must provides helper functions for testing.
package must

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/stretchr/testify/require"
)

type T interface {
	Helper()
	Errorf(format string, args ...interface{})
	FailNow()
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

func DoRequest(t T, client Client, req *http.Request) (*http.Response, []byte) {
	t.Helper()
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer func() { require.NoError(t, resp.Body.Close()) }()
	body := ReadAll(t, resp.Body)
	resp.Body = io.NopCloser(bytes.NewReader(body))
	return resp, body
}

func NewRequest(t T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	require.NoError(t, err)
	return req
}

func Marshal(t T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

func Unmarshal(t T, data []byte, v interface{}) {
	t.Helper()
	require.NoError(t, json.Unmarshal(data, v))
}

func ReadAll(t T, r io.Reader) []byte {
	t.Helper()
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	return b
}
