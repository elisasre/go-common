package sentryutil

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoverWithContext(t *testing.T) {
	tests := []struct {
		name  string
		cause string
		fn    func()
	}{
		{
			name:  "string panic",
			cause: "panic: test panic",
			fn:    func() { panic("test panic") },
		},
		{
			name:  "error panic",
			cause: "panic: error panic",
			fn:    func() { panic(fmt.Errorf("error panic")) },
		},
		{
			name:  "runtime error",
			cause: "panic: runtime error: index out of range",
			fn:    func() { _ = []int{}[1] },
		},
	}

	t.Cleanup(func() {
		output = os.Stdout
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			output = out

			func() {
				ctx := context.Background()
				span := sentry.SpanFromContext(ctx)
				defer RecoverWithContext(ctx, span)
				tt.fn()
			}()

			buf := out.String()
			hasPrefix := strings.HasPrefix(buf, tt.cause)
			require.True(t, hasPrefix, "expected %q to start with %q", buf, tt.cause)
		})
	}
}

func TestGinWrapper(t *testing.T) {
	tests := []struct {
		name string
		fn   func(*gin.RouterGroup, string, ...gin.HandlerFunc) gin.IRoutes
	}{
		{
			name: "GET",
			fn:   GET,
		},
		{
			name: "POST",
			fn:   POST,
		},
		{
			name: "PUT",
			fn:   PUT,
		},
		{
			name: "DELETE",
			fn:   DELETE,
		},
		{
			name: "PATCH",
			fn:   PATCH,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			gr := r.Group("/")
			tt.fn(gr, tt.name, func(c *gin.Context) {
				span := sentry.SpanFromContext(c.Request.Context())
				assert.NotNil(t, span)
			})

			req := httptest.NewRequest(tt.name, "/", nil)
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
		})
	}
}

func TestMakeSpan(t *testing.T) {
	span := MakeSpan(context.Background(), 0)
	assert.NotNil(t, span)
}

func TestMakeTransaction(t *testing.T) {
	ctx, span, hub := MakeTransaction(context.Background(), "test")
	assert.NotNil(t, ctx)
	assert.NotNil(t, span)
	assert.NotNil(t, hub)
}

func TestErrorDoesntPanic(t *testing.T) {
	ctx := context.Background()
	err := fmt.Errorf("test error")

	Error(ctx, err)

	ErrorUnlessIgnored(ctx, err)
	ErrorUnlessIgnored(ctx, nil)
	ErrorUnlessIgnored(ctx, context.Canceled)
	ErrorUnlessIgnored(ctx, context.DeadlineExceeded, context.Canceled, context.DeadlineExceeded)
}
