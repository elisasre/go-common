package tracerprovider_test

import (
	"context"
	"fmt"

	"github.com/elisasre/go-common/v2/service"
	"github.com/elisasre/go-common/v2/service/module/tracerprovider"
	"google.golang.org/grpc/credentials/insecure"
)

func ExampleNew() {
	tp := tracerprovider.New(
		tracerprovider.WithSamplePercentage(42),
		tracerprovider.WithCollector("localhost", 4317, insecure.NewCredentials()),
		tracerprovider.WithContext(context.Background()),
		tracerprovider.WithServiceName("test"),
		tracerprovider.WithProcessor("processor"),
	)
	err := service.Run(service.Modules{tp})
	if err != nil {
		fmt.Println(err)
	}
	// Output: failed to initialize module otel.TracerProvider: otel.TracerProvider option error: invalid processor
}