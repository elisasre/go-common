package errorreportingutil

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"

	"cloud.google.com/go/errorreporting"

	"golang.org/x/oauth2"
	"google.golang.org/api/option"
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
	tokenSource                            oauth2.TokenSource
	onErrFunc                              func(error)
	projectID, serviceName, serviceVersion string
	ignoredErrors                          []error
}

type Opt func(client *errorReporting)

type ErrorClient interface {
	Report(entry errorreporting.Entry)
	Close() error
}

// Set error reporting client.
func WithClient(client ErrorClient) Opt {
	return func(er *errorReporting) {
		er.client = client
	}
}

// Set Google Cloud project ID.
func WithProjectID(projectID string) Opt {
	return func(er *errorReporting) {
		er.projectID = projectID
	}
}

// Set service name.
func WithServiceName(serviceName string) Opt {
	return func(er *errorReporting) {
		er.serviceName = serviceName
	}
}

// Set service version.
func WithServiceVersion(serviceVersion string) Opt {
	return func(er *errorReporting) {
		er.serviceVersion = serviceVersion
	}
}

// List of errors to ignore instead of sending.
func WithIgnoredErrors(ignoredErrors []error) Opt {
	return func(er *errorReporting) {
		er.ignoredErrors = ignoredErrors
	}
}

// Token source to use for authentication with error reporting.
func WithTokenSource(tokenSource oauth2.TokenSource) Opt {
	return func(er *errorReporting) {
		er.tokenSource = tokenSource
	}
}

// Function to call when sending an error report fails. Defaults to logging.
func WithOnError(f func(error)) Opt {
	return func(er *errorReporting) {
		er.onErrFunc = f
	}
}

func (*NoOpClient) Report(entry errorreporting.Entry) {
	slog.Info("received error entry", slog.Any("entry", entry))
}

func (*NoOpClient) Close() error {
	slog.Info("closed noop errorReporter")
	return nil
}

// Report an error unless specifically ignored.
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
func Init(ctx context.Context, opts ...Opt) (*errorReporting, error) {
	er := &errorReporting{}

	for _, o := range opts {
		o(er)
	}

	if er.projectID == "" {
		return nil, ErrProjectIDNotSet
	}

	if er.client == nil {
		clientOpts := make([]option.ClientOption, 0)
		if er.tokenSource != nil {
			clientOpts = append(clientOpts, option.WithTokenSource(er.tokenSource))
		}
		cfg := errorreporting.Config{
			ServiceName:    er.serviceName,
			ServiceVersion: er.serviceVersion,
		}
		if er.onErrFunc != nil {
			cfg.OnError = er.onErrFunc
		}

		c, err := errorreporting.NewClient(ctx, er.projectID, cfg, clientOpts...)
		if err != nil {
			return nil, err
		}
		er.client = c
	}

	mu.Lock()
	errorReporter = er
	mu.Unlock()

	return er, nil
}

// Flush pending entries and close the client.
func Close() error {
	mu.RLock()
	defer mu.RUnlock()
	return errorReporter.client.Close()
}
