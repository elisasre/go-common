package errorreportingutil_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/errorreporting"
	"github.com/elisasre/go-common/v2/auth"
	eutil "github.com/elisasre/go-common/v2/errorreportingutil"
	"github.com/gin-gonic/gin"

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
	testErr := errors.New("test error")
	wrappedErr := errors.Join(baseErr, errors.New("additional context"))

	tests := []struct {
		name            string
		ignoredErrors   []error
		reportError     error
		expectedEntries []errorreporting.Entry
	}{
		{
			name:          "reports when not ignored",
			ignoredErrors: []error{ignoredErr1},
			reportError:   testErr,
			expectedEntries: []errorreporting.Entry{
				{Error: testErr},
			},
		},
		{
			name:          "ignores exact matching error",
			ignoredErrors: []error{ignoredErr1},
			reportError:   ignoredErr1,
		},
		{
			name:          "ignores wrapped error",
			ignoredErrors: []error{baseErr},
			reportError:   wrappedErr,
		},
		{
			name:          "ignores first error in list",
			ignoredErrors: []error{ignoredErr1, ignoredErr2},
			reportError:   ignoredErr1,
		},
		{
			name:          "ignores second error in list",
			ignoredErrors: []error{ignoredErr1, ignoredErr2},
			reportError:   ignoredErr2,
		},
		{
			name:          "reports non-ignored with multiple ignored errors",
			ignoredErrors: []error{ignoredErr1, ignoredErr2},
			reportError:   testErr,
			expectedEntries: []errorreporting.Entry{
				{Error: testErr},
			},
		},
		{
			name:          "handles nil error",
			ignoredErrors: []error{},
			reportError:   nil,
			expectedEntries: []errorreporting.Entry{
				{Error: nil},
			},
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

			assert.EqualValues(t, tt.expectedEntries, mock.reported)
		})
	}
}

func TestRecover_StringPanic(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	assert.Panics(t, func() {
		defer eutil.Recover()
		panic("test panic")
	})

	require.Len(t, mock.reported, 1)
	assert.Contains(t, mock.reported[0].Error.Error(), "test panic")
	assert.NotEmpty(t, mock.reported[0].Stack, "should capture stack trace")
}

func TestRecover_ErrorPanic(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	testErr := errors.New("error panic")
	assert.PanicsWithError(t, "error panic", func() {
		defer eutil.Recover()
		panic(testErr)
	})

	require.Len(t, mock.reported, 1)
	assert.Equal(t, testErr, mock.reported[0].Error)
	assert.NotEmpty(t, mock.reported[0].Stack)
	stackStr := string(mock.reported[0].Stack)
	assert.NotContains(t, stackStr, "errorreportingutil", "stack trace should not contain errorreportingutil frames")
}

func TestRecover_NoPanic(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		defer eutil.Recover()
		// No panic
	})

	assert.Empty(t, mock.reported, "should not report anything when no panic occurs")
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
	assert.NotEmpty(t, mock.reported[0].Stack, "should capture stack trace when not provided")

	stackStr := string(mock.reported[0].Stack)
	assert.NotContains(t, stackStr, "errorreportingutil", "stack trace should not contain errorreportingutil frames")
}

func TestReport_WithStackProvided(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	testErr := errors.New("test error")
	providedStack := []byte("custom stack trace")
	eutil.Report(errorreporting.Entry{Error: testErr, Stack: providedStack})

	require.Len(t, mock.reported, 1)
	assert.Equal(t, "test error", mock.reported[0].Error.Error())
	assert.Equal(t, providedStack, mock.reported[0].Stack, "should preserve provided stack trace")
}

func TestError_StackTrace(t *testing.T) {
	mock := &mockErrorClient{}
	_, err := eutil.Init(
		context.Background(),
		eutil.WithProjectID("12345"),
		eutil.WithClient(mock),
	)
	require.NoError(t, err)

	testErr := errors.New("test error")
	eutil.Error(context.Background(), testErr)

	require.Len(t, mock.reported, 1)
	entry := mock.reported[0]
	assert.NotEmpty(t, entry.Stack, "should capture stack trace")

	stackStr := string(entry.Stack)
	assert.NotContains(t, stackStr, "errorreportingutil", "stack trace should not contain errorreportingutil frames")
	assert.Contains(t, stackStr, "testing", "stack trace should contain testing frames")
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	testErr := errors.New("test error")

	tests := []struct {
		name            string
		setupContext    func() context.Context
		err             error
		expectedReqPath string
		expectedEntry   errorreporting.Entry
	}{
		{
			name: "nil context",
			setupContext: func() context.Context {
				return nil
			},
			err: testErr,
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				Req:   nil,
				User:  "",
			},
		},
		{
			name: "regular context (non-gin)",
			setupContext: func() context.Context { //nolint:gocritic
				return context.Background()
			},
			err: testErr,
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				Req:   nil,
				User:  "",
			},
		},
		{
			name: "gin context with request",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodGet, "/test/path", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				return c
			},
			err:             testErr,
			expectedReqPath: "/test/path",
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				User:  "",
			},
		},
		{
			name: "gin context with user",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodPost, "/api/endpoint", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				email := "test@example.com"
				c.Set("user", &auth.User{Email: &email})
				return c
			},
			err:             testErr,
			expectedReqPath: "/api/endpoint",
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				User:  "test@example.com",
			},
		},
		{
			name: "gin context without user",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodGet, "/public", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				return c
			},
			err:             testErr,
			expectedReqPath: "/public",
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				User:  "",
			},
		},
		{
			name: "gin context with user without email",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				c.Set("user", &auth.User{Email: nil})
				return c
			},
			err:             testErr,
			expectedReqPath: "/test",
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				User:  "",
			},
		},
		{
			name: "gin context with non-user type in context",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				c.Set("user", "not a user struct")
				return c
			},
			err:             testErr,
			expectedReqPath: "/test",
			expectedEntry: errorreporting.Entry{
				Error: testErr,
				User:  "",
			},
		},
		{
			name: "nil error with gin context",
			setupContext: func() context.Context {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(w)
				c.Request = req
				return c
			},
			err:             nil,
			expectedReqPath: "/test",
			expectedEntry: errorreporting.Entry{
				Error: nil,
				User:  "",
			},
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

			ctx := tt.setupContext()
			eutil.Error(ctx, tt.err)

			require.Len(t, mock.reported, 1)
			entry := mock.reported[0]

			assert.Equal(t, tt.expectedEntry.Error, entry.Error)
			assert.Equal(t, tt.expectedEntry.User, entry.User)

			if tt.expectedReqPath != "" {
				require.NotNil(t, entry.Req)
				assert.Equal(t, tt.expectedReqPath, entry.Req.URL.Path)
			} else {
				assert.Nil(t, entry.Req)
			}
		})
	}
}
