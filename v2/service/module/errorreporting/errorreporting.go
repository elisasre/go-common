// Package errorreporting provides a service runner module that ensures events are flushed on service shutdown.
package errorreporting

import (
	"errors"
)

type ErrorReportingClient interface {
	Close() error
}

type ErrorReporting struct {
	client ErrorReportingClient
}

type Opt func(*ErrorReporting)

func WithClient(c ErrorReportingClient) Opt {
	return func(s *ErrorReporting) {
		s.client = c
	}
}

func New(opts ...Opt) *ErrorReporting {
	er := &ErrorReporting{}

	for _, o := range opts {
		o(er)
	}

	return er
}

func (er *ErrorReporting) Init() error {
	if er.client == nil {
		return errors.New("error reporting client not provided")
	}

	return nil
}

func (er *ErrorReporting) Run() error {
	return nil
}

func (er *ErrorReporting) Stop() error {
	return er.client.Close()
}

func (er *ErrorReporting) Name() string {
	return "errorreporting.Errorreporting"
}
