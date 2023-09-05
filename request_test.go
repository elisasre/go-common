package common

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleMakeRequest() {
	// retry once in second, maximum retries 2 times
	backoff := Backoff{
		Duration:   1 * time.Second,
		MaxRetries: 2,
	}

	type Out struct {
		Message string `json:"message"`
	}
	out := Out{}
	client := &http.Client{}
	ctx := context.Background()
	body, err := MakeRequest(
		ctx,
		HTTPRequest{
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
	_, err = MakeRequest(
		ctx,
		HTTPRequest{
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
	backoff := Backoff{
		Duration:   100 * time.Millisecond,
		MaxRetries: 1,
	}

	client := &MockClient{
		DoFunc: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"hello":"world"}`)),
			}, nil
		},
	}

	ctx := context.Background()
	body, err := MakeRequest(
		ctx,
		HTTPRequest{
			URL:    "http://foobar",
			Method: "GET",
			OKCode: []int{200},
		},
		nil,
		client,
		backoff,
	)
	require.NoError(t, err)
	assert.Equal(t, `{"hello":"world"}`, body.Body)
	assert.Equal(t, 200, body.StatusCode)
}
