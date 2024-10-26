package tracerprovider_test

import (
	"context"
	"errors"
	"testing"

	"github.com/elisasre/go-common/v2/service/module/tracerprovider"
	"github.com/hashicorp/go-multierror"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/credentials/insecure"
)

func TestTracerProviderGRPC(t *testing.T) {
	tp := tracerprovider.New(
		tracerprovider.WithSamplePercentage(42),
		tracerprovider.WithGRPCExporter("localhost:4317", insecure.NewCredentials()),
		tracerprovider.WithContext(context.Background()),
		tracerprovider.WithServiceName("test"),
		tracerprovider.WithEnvironment("development"),
		tracerprovider.WithProcessor(tracerprovider.ProcessorBatch),
	)
	require.NoError(t, tp.Init())
	wg := &multierror.Group{}
	wg.Go(tp.Run)
	require.NoError(t, tp.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "otel.TracerProvider", tp.Name())
}

func TestTracerProviderHTTP(t *testing.T) {
	tp := tracerprovider.New(
		tracerprovider.WithSamplePercentage(42),
		tracerprovider.WithHTTPExporter("localhost:4318", "ApiKey verysecret", true),
		tracerprovider.WithContext(context.Background()),
		tracerprovider.WithServiceName("test"),
		tracerprovider.WithEnvironment("development"),
		tracerprovider.WithProcessor(tracerprovider.ProcessorBatch),
	)
	require.NoError(t, tp.Init())
	wg := &multierror.Group{}
	wg.Go(tp.Run)
	require.NoError(t, tp.Stop())
	require.NoError(t, wg.Wait().ErrorOrNil())
	require.Equal(t, "otel.TracerProvider", tp.Name())
}

func TestTracerProviderInitErrors(t *testing.T) {
	errOpt := errors.New("otel.TracerProvider option error")

	tests := []struct {
		name        string
		tp          *tracerprovider.TracerProvider
		expectedErr error
	}{
		{
			name:        "ErrOpt",
			tp:          tracerprovider.New(func(tp *tracerprovider.TracerProvider) error { return errOpt }),
			expectedErr: errOpt,
		},
		{
			name:        "ErrSamplePercentageOverRange",
			tp:          tracerprovider.New(tracerprovider.WithSamplePercentage(110)),
			expectedErr: tracerprovider.ErrInvalidSamplePercentage,
		},
		{
			name:        "ErrSamplePercentageUnderRange",
			tp:          tracerprovider.New(tracerprovider.WithSamplePercentage(-1)),
			expectedErr: tracerprovider.ErrInvalidSamplePercentage,
		},
		{
			name:        "ErrInvalidProcessor",
			tp:          tracerprovider.New(tracerprovider.WithProcessor("foo")),
			expectedErr: tracerprovider.ErrInvalidProcessor,
		},
		{
			name:        "ErrInvalidToken",
			tp:          tracerprovider.New(tracerprovider.WithHTTPExporter("localhost:4318", "token", true)),
			expectedErr: tracerprovider.ErrInvalidToken,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := tc.tp.Init()
			require.ErrorIs(t, err, tc.expectedErr)
		})
	}
}
