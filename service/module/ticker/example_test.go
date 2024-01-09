package ticker_test

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/elisasre/go-common/service"
	"github.com/elisasre/go-common/service/module/ticker"
)

func ExampleNew() {
	t := ticker.New(
		ticker.WithInterval(time.Second),
		ticker.WithFunc(func() error {
			slog.Info("Hello from ticker")
			return errors.New("ticker error")
		}),
	)

	err := service.Run(service.Modules{t})
	if err != nil {
		fmt.Println(err)
	}
	// Output: 1 error occurred:
	//	* failed to run module ticker.Ticker: ticker error
}
