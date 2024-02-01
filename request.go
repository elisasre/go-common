package common

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// HTTPClient allows inserting either *http.Client or mock client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPRequest ...
type HTTPRequest struct {
	Method      string
	URL         string
	Body        []byte
	Cookies     []*http.Cookie
	Headers     map[string]string
	OKCode      []int
	Unmarshaler func(data []byte, v any) error
}

// HTTPResponse ...
type HTTPResponse struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
}

// Backoff contains struct for retrying strategy.
type Backoff struct {
	// The initial duration.
	Duration time.Duration
	// The remaining number of iterations in which the duration
	// parameter may change. If not positive, the duration is not
	// changed.
	MaxRetries int
}

// MakeRequest ...
func MakeRequest(
	ctx context.Context,
	request HTTPRequest,
	output interface{},
	client HTTPClient,
	backoff Backoff,
) (*HTTPResponse, error) {
	httpresp := &HTTPResponse{}
	if request.Unmarshaler == nil {
		request.Unmarshaler = json.Unmarshal
	}

	err := SleepUntil(backoff, func() (bool, error) {
		httpreq, err := http.NewRequestWithContext(ctx, request.Method, request.URL, nil)
		if err != nil {
			slog.Error("creating http request failed",
				slog.String("method", request.Method),
				slog.String("url", request.URL),
				slog.String("error", err.Error()),
			)
			return false, err
		}
		if len(request.Body) > 0 {
			httpreq.Body = io.NopCloser(bytes.NewReader(request.Body))
		}

		for k, v := range request.Headers {
			httpreq.Header.Add(k, v)
		}

		for _, cookie := range request.Cookies {
			httpreq.AddCookie(cookie)
		}

		resp, err := client.Do(httpreq)
		if err != nil {
			slog.Error("http request failed",
				slog.String("method", request.Method),
				slog.String("url", request.URL),
				slog.String("error", err.Error()),
			)
			if errors.Is(err, context.DeadlineExceeded) {
				return true, err
			}
			return false, err
		}
		defer resp.Body.Close()
		httpresp.StatusCode = resp.StatusCode
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		httpresp.Body = responseBody
		httpresp.Headers = resp.Header
		if ContainsInteger(request.OKCode, resp.StatusCode) {
			if output != nil {
				err = request.Unmarshaler(httpresp.Body, &output)
				if err != nil {
					return true, fmt.Errorf("could not marshal %w", err)
				}
			}
			return true, nil
		}

		msg := "retrying"
		rtn := false
		if resp.StatusCode == http.StatusTooManyRequests {
			msg = "too many requests"
			rtn = true
			err = fmt.Errorf("rate limit exceeded")
		}
		slog.Error(msg,
			slog.Int("status_code", resp.StatusCode),
			slog.String("method", request.Method),
			slog.String("url", request.URL),
			slog.String("body", string(responseBody)),
		)
		return rtn, err
	})
	return httpresp, err
}

// MockClient is helper client for mock tests.
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do executes the HTTPClient interface Do function.
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}
