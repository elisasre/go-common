package errorreportingutil_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cloud.google.com/go/errorreporting"

	eutil "github.com/elisasre/go-common/v2/errorreportingutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockErrorClient struct {
	reported []errorreporting.Entry
	closed   bool
}

func (m *mockErrorClient) Report(entry errorreporting.Entry) {
	m.reported = append(m.reported, entry)
}

func (m *mockErrorClient) Close() error {
	m.closed = true
	return nil
}

func TestReportUnlessIgnored(t *testing.T) {
	baseErr := errors.New("base error")
	ignoredErr1 := errors.New("ignored error 1")
	ignoredErr2 := errors.New("ignored error 2")

	tests := []struct {
		name           string
		ignoredErrors  []error
		reportError    error
		expectReported bool
		expectedMsg    string
	}{
		{
			name:           "reports when not ignored",
			ignoredErrors:  []error{ignoredErr1},
			reportError:    errors.New("test error"),
			expectReported: true,
			expectedMsg:    "test error",
		},
		{
			name:           "ignores exact matching error",
			ignoredErrors:  []error{ignoredErr1},
			reportError:    ignoredErr1,
			expectReported: false,
		},
		{
			name:           "ignores wrapped error",
			ignoredErrors:  []error{baseErr},
			reportError:    errors.Join(baseErr, errors.New("additional context")),
			expectReported: false,
		},
		{
			name:           "ignores first error in list",
			ignoredErrors:  []error{ignoredErr1, ignoredErr2},
			reportError:    ignoredErr1,
			expectReported: false,
		},
		{
			name:           "ignores second error in list",
			ignoredErrors:  []error{ignoredErr1, ignoredErr2},
			reportError:    ignoredErr2,
			expectReported: false,
		},
		{
			name:           "reports non-ignored with multiple ignored errors",
			ignoredErrors:  []error{ignoredErr1, ignoredErr2},
			reportError:    errors.New("test error"),
			expectReported: true,
			expectedMsg:    "test error",
		},
		{
			name:           "handles nil error",
			ignoredErrors:  []error{},
			reportError:    nil,
			expectReported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockErrorClient{}
			_, err := eutil.Init(
				context.Background(),
				eutil.WithProjectID("12345"),
				eutil.WithClient(mock),
				eutil.WithIgnoredErrors(tt.ignoredErrors),
			)
			require.NoError(t, err)

			eutil.ReportUnlessIgnored(errorreporting.Entry{Error: tt.reportError})

			if tt.expectReported {
				require.Len(t, mock.reported, 1)
				if tt.reportError != nil {
					assert.Equal(t, tt.expectedMsg, mock.reported[0].Error.Error())
				} else {
					assert.Nil(t, mock.reported[0].Error)
				}
			} else {
				assert.Empty(t, mock.reported, "should not report ignored errors")
			}
		})
	}
}

func TestRecover(t *testing.T) {
	testErr := errors.New("error panic")

	tests := []struct {
		name             string
		panicValue       any
		expectPanic      bool
		expectReported   bool
		validateError    func(t *testing.T, err error)
		validateStack    func(t *testing.T, stack []byte)
		validateOutput   func(t *testing.T, output string)
		expectedPanicMsg string
	}{
		{
			name:           "string panic",
			panicValue:     "test panic",
			expectPanic:    true,
			expectReported: true,
			validateError: func(t *testing.T, err error) {
				assert.Contains(t, err.Error(), "test panic")
			},
			validateStack: func(t *testing.T, stack []byte) {
				assert.NotEmpty(t, stack, "should capture stack trace")
			},
			validateOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "panic: test panic")
				lines := strings.Split(output, "\n")
				assert.True(t, len(lines) > 1, "should have multiple lines including stack trace")
			},
		},
		{
			name:           "error panic",
			panicValue:     testErr,
			expectPanic:    true,
			expectReported: true,
			validateError: func(t *testing.T, err error) {
				assert.Equal(t, testErr, err)
			},
			validateStack: func(t *testing.T, stack []byte) {
				assert.NotEmpty(t, stack)
				stackStr := string(stack)
				assert.Contains(t, stackStr, "errorreportingutil", "stack trace should contain package name")
			},
			validateOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "panic: error panic")
			},
			expectedPanicMsg: "error panic",
		},
		{
			name:           "no panic",
			panicValue:     nil,
			expectPanic:    false,
			expectReported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockErrorClient{}

			_, err := eutil.Init(
				context.Background(),
				eutil.WithProjectID("12345"),
				eutil.WithClient(mock),
			)
			require.NoError(t, err)

			if tt.expectPanic {
				if tt.expectedPanicMsg != "" {
					assert.PanicsWithError(t, tt.expectedPanicMsg, func() {
						defer eutil.Recover()
						panic(tt.panicValue)
					})
				} else {
					assert.Panics(t, func() {
						defer eutil.Recover()
						panic(tt.panicValue)
					})
				}
			} else {
				assert.NotPanics(t, func() {
					defer eutil.Recover()
					// No panic
				})
			}

			if tt.expectReported {
				require.Len(t, mock.reported, 1)
				if tt.validateError != nil {
					tt.validateError(t, mock.reported[0].Error)
				}
				if tt.validateStack != nil {
					tt.validateStack(t, mock.reported[0].Stack)
				}
			} else {
				assert.Empty(t, mock.reported, "should not report anything when no panic occurs")
			}
		})
	}
}

func TestReport(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	testErr := errors.New("test error")
	eutil.Report(errorreporting.Entry{Error: testErr})

	require.Len(t, mock.reported, 1)
	assert.Equal(t, "test error", mock.reported[0].Error.Error())
}

func TestClose(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	err = eutil.Close()

	assert.NoError(t, err)
	assert.True(t, mock.closed, "errorReporter should be closed")
}
