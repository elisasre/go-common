package httputil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// HTTPClient allows inserting either *http.Client or mock client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Request contains all relevant data for making http requst.
type Request struct {
	Method                    string
	URL                       string
	Body                      []byte
	Cookies                   []*http.Cookie
	Headers                   map[string]string
	OKCode                    []int
	Unmarshaler               func(data []byte, v any) error
	ContinueOnContextDeadline bool
}

// Response contains basic fields extracted from http.Response.
type Response struct {
	Body       []byte
	StatusCode int
	Headers    http.Header
}

// Backoff slices.Contains struct for retrying strategy.
type Backoff struct {
	// The initial duration.
	Duration time.Duration
	// Maximum number of tries.
	MaxTries int
}

// MakeRequest is hihg level wrapper for http.Do with added functionality like retries and automatic response parsing.
func MakeRequest(ctx context.Context, request Request, output interface{}, client HTTPClient, backoff Backoff) (*Response, error) {
	httpresp := &Response{}
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
			if !request.ContinueOnContextDeadline && errors.Is(err, context.DeadlineExceeded) {
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
		if slices.Contains(request.OKCode, resp.StatusCode) {
			if output != nil {
				err = request.Unmarshaler(httpresp.Body, &output)
				if err != nil {
					return true, fmt.Errorf("could not marshal %w", err)
				}
			}
			return true, nil
		}

		l := slog.With(
			slog.Int("status_code", resp.StatusCode),
			slog.String("method", request.Method),
			slog.String("url", request.URL),
			slog.String("body", string(responseBody)),
		)

		rtn := false
		if resp.StatusCode == http.StatusTooManyRequests {
			rtn = true
			err = fmt.Errorf("rate limit exceeded")
			l.Error("too many requests")
		}

		l.Error("retrying")
		return rtn, err
	})
	return httpresp, err
}

// ErrTimeout is returned if SleepUntil condition isn't met.
var ErrTimeout = errors.New("timed out waiting for the condition")

// ConditionFunc returns true if the condition is satisfied, or an error
// if the loop should be aborted.
type ConditionFunc func() (done bool, err error)

// SleepUntil waits for condition to succeeds.
func SleepUntil(backoff Backoff, condition ConditionFunc) error {
	var err error
	for backoff.MaxTries > 0 {
		var ok bool
		if ok, err = condition(); ok {
			return err
		}
		if backoff.MaxTries == 1 {
			break
		}
		backoff.MaxTries--
		time.Sleep(backoff.Duration)

	}
	if err != nil {
		return err
	}
	return ErrTimeout
}

// MockClient is helper client for mock tests.
type MockClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

// Do executes the HTTPClient interface Do function.
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return m.DoFunc(req)
}
