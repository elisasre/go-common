package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/wait"
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

// MakeRequest ...
func MakeRequest(request HTTPRequest, output interface{}, client *http.Client, backoff wait.Backoff) (*HTTPResponse, error) {
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
