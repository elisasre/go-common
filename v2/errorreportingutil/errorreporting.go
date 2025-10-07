// Package errorreportingutil provides a utility to send errors to Google Cloud error reporting.
package errorreportingutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
	"sync"

	"github.com/elisasre/go-common/v2"
	"github.com/elisasre/go-common/v2/auth"

	"github.com/gin-gonic/gin"

	"cloud.google.com/go/errorreporting"
)

var (
	mu            sync.RWMutex
	errorReporter = &errorReporting{
		client:         &NoOpClient{},
		projectID:      "",
		serviceName:    "",
		serviceVersion: "",
		ignoredErrors:  nil,
	}
	ErrProjectIDNotSet = errors.New("project id not set")
)

type NoOpClient struct{}

type errorReporting struct {
	client                                 ErrorClient
	onErrFunc                              func(error)
	projectID, serviceName, serviceVersion string
	ignoredErrors                          []error
}

type Opt func(client *errorReporting)

type ErrorClient interface {
	Report(entry errorreporting.Entry)
	Close() error
}

// WithClient sets the client to use for error reporting.
func WithClient(client ErrorClient) Opt {
	return func(er *errorReporting) {
		er.client = client
	}
}

// WithProjectID sets Google Cloud project ID.
func WithProjectID(projectID string) Opt {
	return func(er *errorReporting) {
		er.projectID = projectID
	}
}

// WithServiceName sets the service name.
func WithServiceName(serviceName string) Opt {
	return func(er *errorReporting) {
		er.serviceName = serviceName
	}
}

// WithServiceVersion sets the service version.
func WithServiceVersion(serviceVersion string) Opt {
	return func(er *errorReporting) {
		er.serviceVersion = serviceVersion
	}
}

// WithIgnoredErrors provides a list of errors to ignore instead of sending.
func WithIgnoredErrors(ignoredErrors []error) Opt {
	return func(er *errorReporting) {
		er.ignoredErrors = ignoredErrors
	}
}

// WithOnError sets the function to call when sending an error report fails. Defaults to logging.
func WithOnError(f func(error)) Opt {
	return func(er *errorReporting) {
		er.onErrFunc = f
	}
}

func (*NoOpClient) Report(entry errorreporting.Entry) {
	slog.Warn("received error entry", slog.Any("entry", entry))
}

func (*NoOpClient) Close() error {
	slog.Info("closed noop errorReporter")
	return nil
}

func ReportUnlessIgnored(entry errorreporting.Entry) {
	mu.RLock()
	defer mu.RUnlock()

	for _, ignoredError := range errorReporter.ignoredErrors {
		if errors.Is(entry.Error, ignoredError) {
			slog.Debug("ignoring error", slog.Any("entry", entry))
			return
		}
	}

	errorReporter.client.Report(entry)
}

func getUserFromCtx(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if gCtx, ok := ctx.(*gin.Context); ok {
		if u, ok := gCtx.Get("user"); ok {
			if user, ok := u.(*auth.User); ok {
				return common.ValOrZero(user.Email)
			}
		}
	}
	return ""
}

// Error reports an error and tries to populate request and user information from context.
// Request parsing works if passed context is a Gin context.
func Error(ctx context.Context, err error) {
	var req *http.Request
	if ctx != nil {
		if gCtx, ok := ctx.(*gin.Context); ok {
			req = gCtx.Request
		}
	}

	user := getUserFromCtx(ctx)

	Report(errorreporting.Entry{
		Error: err,
		Req:   req,
		User:  user,
	})
}

// Report an error unless specifically ignored.
func Report(entry errorreporting.Entry) {
	ReportUnlessIgnored(entry)
}

// Recover panic, send error report and panic again.
func Recover() {
	if r := recover(); r != nil {
		var err error
		if e, ok := r.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic: %v", r)
		}

		stack := debug.Stack()

		Report(errorreporting.Entry{
			Error: err,
			Stack: stack,
		})

		panic(r)
	}
}

// Init error reporting and set up GPC error reporting instead of the default noop client.
func Init(ctx context.Context, opts ...Opt) (*errorreporting.Client, error) {
	er := &errorReporting{}

	for _, o := range opts {
		o(er)
	}

	if er.projectID == "" {
		return nil, ErrProjectIDNotSet
	}

	var c *errorreporting.Client
	if er.client == nil {
		cfg := errorreporting.Config{
			ServiceName:    er.serviceName,
			ServiceVersion: er.serviceVersion,
		}
		if er.onErrFunc != nil {
			cfg.OnError = er.onErrFunc
		}

		var err error
		c, err = errorreporting.NewClient(ctx, er.projectID, cfg)
		if err != nil {
			return nil, err
		}
		er.client = c
	}

	mu.Lock()
	errorReporter = er
	mu.Unlock()

	return c, nil
}

// Flush pending entries and close the client.
func Close() error {
	mu.RLock()
	defer mu.RUnlock()
	return errorReporter.client.Close()
}
