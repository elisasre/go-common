package tracerprovider_test

import (
	"fmt"

	"github.com/elisasre/go-common/v2/service"
	"github.com/elisasre/go-common/v2/service/module/tracerprovider"
	"google.golang.org/grpc/credentials/insecure"
)

func ExampleNew() {
	tp := tracerprovider.New(
		tracerprovider.WithSamplePercentage(42),
		tracerprovider.WithCollector("localhost", 4317, insecure.NewCredentials()),
		tracerprovider.WithServiceName("test"),
		tracerprovider.WithProcessor("batch"),
	)
	err := service.Run(service.Modules{tp})
	if err != nil {
		fmt.Println(err)
	}
	// Output:
}
