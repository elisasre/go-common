package must_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elisasre/go-common/v2/must"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := noFail1(t, func(mt *mockT) []byte { return must.ReadAll(mt, r.Body) })
		assert.Equal(t, `"ping"`, string(data))

		_, err := fmt.Fprint(w, `"pong"`)
		assert.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	payload := noFail1(t, func(mt *mockT) []byte {
		return must.Marshal(mt, "ping")
	})

	req := noFail1(t, func(mt *mockT) *http.Request {
		return must.NewRequest(mt, "GET", srv.URL, bytes.NewReader(payload))
	})

	resp, body := noFail2(t, func(mt *mockT) (*http.Response, []byte) {
		return must.DoRequest(mt, srv.Client(), req)
	})
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var s string
	noFail(t, func(mt *mockT) { must.Unmarshal(mt, body, &s) })
	assert.Equal(t, "pong", s)

	data := must.ReadAll(t, resp.Body)
	require.Equal(t, body, data)
}

func TestMustEncodeJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	noFail(t, func(mt *mockT) { must.EncodeJSON(mt, buf, "hello") })
	assert.JSONEq(t, `"hello"`, buf.String())
}

func TestMustDecodeJSON(t *testing.T) {
	buf := bytes.NewReader([]byte(`"world"`))
	var s string
	noFail(t, func(mt *mockT) { must.DecodeJSON(mt, buf, &s) })
	assert.Equal(t, "world", s)
}

type mockT struct {
	failed bool
	msg    string
}

func (m *mockT) Helper()                              {}
func (m *mockT) Errorf(f string, args ...interface{}) { m.msg = fmt.Sprintf(f, args...) }
func (m *mockT) FailNow()                             { m.failed = true }

func noFail(t *testing.T, f func(mt *mockT)) {
	t.Helper()
	mt := &mockT{}
	f(mt)
	assert.False(t, mt.failed)
	assert.Empty(t, mt.msg)
}

func noFail1[T1 any](t *testing.T, f func(mt *mockT) T1) T1 {
	var v1 T1
	noFail(t, func(mt *mockT) { v1 = f(mt) })
	return v1
}

func noFail2[T1, T2 any](t *testing.T, f func(mt *mockT) (T1, T2)) (T1, T2) {
	var v1 T1
	var v2 T2
	noFail(t, func(mt *mockT) { v1, v2 = f(mt) })
	return v1, v2
}
