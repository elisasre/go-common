package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/golang/glog"
)

// HTTPRequest ...
type HTTPRequest struct {
	Method  string
	URL     string
	Body    []byte
	Cookies []*http.Cookie
	Headers map[string]string
	OKCode  []int
}

// HTTPResponse ...
type HTTPResponse struct {
	Body       []byte
	StatusCode int
}

// Backoff contains struct for retrying strategy
type Backoff struct {
	// The initial duration.
	Duration time.Duration
	// The remaining number of iterations in which the duration
	// parameter may change. If not positive, the duration is not
	// changed.
	MaxRetries int
}

// MakeRequest ...
func MakeRequest(request HTTPRequest, output interface{}, client *http.Client, backoff Backoff) (*HTTPResponse, error) {
	httpresp := &HTTPResponse{}
	err := SleepUntil(backoff, func() (bool, error) {
		httpreq, err := http.NewRequest(request.Method, request.URL, nil)
		if err != nil {
			glog.Errorf("Request error from [%s] %s: %v", request.Method, request.URL, err)
			return false, err
		}
		if len(request.Body) > 0 {
			httpreq.Body = ioutil.NopCloser(bytes.NewReader(request.Body))
		}

		for k, v := range request.Headers {
			httpreq.Header.Add(k, v)
		}

		for _, cookie := range request.Cookies {
			httpreq.AddCookie(cookie)
		}

		resp, err := client.Do(httpreq)
		if err != nil {
			glog.Errorf("Do request error from [%s] %s: %v", request.Method, request.URL, err)
			return false, err
		}
		defer resp.Body.Close()
		httpresp.StatusCode = resp.StatusCode
		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return false, err
		}
		httpresp.Body = responseBody
		if ContainsInteger(request.OKCode, resp.StatusCode) {
			err = json.Unmarshal(httpresp.Body, &output)
			if err != nil {
				return true, fmt.Errorf("could not marshal as json %w", err)
			}
			return true, nil
		}
		err = fmt.Errorf("got http code %v from [%s] %s: %s... retrying",
			resp.StatusCode, request.Method, request.URL, responseBody)
		glog.Error(err)
		return false, err
	})
	return httpresp, err
}
