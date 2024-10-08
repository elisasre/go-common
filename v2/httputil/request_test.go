package httputil_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/elisasre/go-common/v2/httputil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleMakeRequest() {
	// retry once in second, maximum retries 2 times
	backoff := httputil.Backoff{
		Duration: 1 * time.Second,
		MaxTries: 2,
	}

	type Out struct {
		Message string `json:"message"`
	}
	out := Out{}
	client := &http.Client{}
	ctx := context.Background()
	body, err := httputil.MakeRequest(
		ctx,
		httputil.Request{
			URL:    "https://ingress-api.csf.elisa.fi/healthz",
			Method: "GET",
			OKCode: []int{200},
		},
		&out,
		client,
		backoff,
	)

	fmt.Printf("%s\n%s\n%d\n%v\n", out.Message, body.Body, body.StatusCode, err)

	ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()
	_, err = httputil.MakeRequest(
		ctx,
		httputil.Request{
			URL:    "https://ingress-api.csf.elisa.fi/healthz",
			Method: "GET",
			OKCode: []int{200},
		},
		&out,
		client,
		backoff,
	)

	fmt.Printf("%v", err)
	// Output: pong
	// {"message":"pong","error":""}
	// 200
	// <nil>
	// Get "https://ingress-api.csf.elisa.fi/healthz": context deadline exceeded
}

func TestMakeRequestMock(t *testing.T) {
	backoff := httputil.Backoff{
		Duration: 100 * time.Millisecond,
		MaxTries: 1,
	}

	helloWorld := `{"hello":"world"}`

	client := &httputil.MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(helloWorld)),
			}, nil
		},
	}

	ctx := context.Background()
	body, err := httputil.MakeRequest(
		ctx,
		httputil.Request{
			URL:    "http://foobar",
			Method: "GET",
			OKCode: []int{200},
		},
		nil,
		client,
		backoff,
	)
	require.NoError(t, err)
	assert.Equal(t, helloWorld, string(body.Body))
	assert.Equal(t, 200, body.StatusCode)
}

func TestBearerTokenFileMock(t *testing.T) {
	backoff := httputil.Backoff{
		Duration: 100 * time.Millisecond,
		MaxTries: 1,
	}

	file, err := os.CreateTemp("", "")
	require.NoError(t, err)
	defer os.Remove(file.Name())

	_, err = file.WriteString("mytoken")
	require.NoError(t, err)

	helloWorld := `{"hello":"world"}`

	client := &httputil.MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			authz := req.Header.Get("Authorization")
			if authz != "Bearer mytoken" {
				return &http.Response{
					StatusCode: 401,
					Body:       io.NopCloser(strings.NewReader("Bearer token not found or invalid")),
				}, nil
			}
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(helloWorld)),
			}, nil
		},
	}

	ctx := context.Background()
	body, err := httputil.MakeRequest(
		ctx,
		httputil.Request{
			BearerTokenFile: file.Name(),
			URL:             "http://foobar",
			Method:          "GET",
			OKCode:          []int{200},
		},
		nil,
		client,
		backoff,
	)
	require.NoError(t, err)
	assert.Equal(t, helloWorld, string(body.Body))
	assert.Equal(t, 200, body.StatusCode)
}
