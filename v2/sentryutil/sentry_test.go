package sentryutil

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/getsentry/sentry-go"
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
