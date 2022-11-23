package common

import (
	"fmt"
	"net/http"
	"time"
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
	body, err := MakeRequest(
		HTTPRequest{
			URL:    "https://ingress-api.csf.elisa.fi/healthz",
			Method: "GET",
			OKCode: []int{200},
		},
		&out,
		client,
		backoff,
	)

	fmt.Printf("%s\n%s\n%d\n%v", out.Message, body.Body, body.StatusCode, err)

	// Output: pong
	// {"message":"pong","error":""}
	// 200
	// <nil>
}
