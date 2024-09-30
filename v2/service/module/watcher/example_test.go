package watcher_test

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/elisasre/go-common/v2/service"
	"github.com/elisasre/go-common/v2/service/module/ticker"
	"github.com/elisasre/go-common/v2/service/module/watcher"
)

func ExampleNew() {
	tmpFile, err := os.CreateTemp("", testFilePattern)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer os.Remove(tmpFile.Name())

	t := ticker.New(
		ticker.WithInterval(time.Second),
		ticker.WithFunc(func() error {
			_, err = tmpFile.Write([]byte("updated"))
			if err != nil {
				return err
			}
			return nil
		}),
	)

	w := watcher.New(
		watcher.WithFilename(tmpFile.Name()),
		watcher.WithFunc(func() error {
			slog.Info("Hello from watcher")
			return errors.New("watcher error")
		}),
	)

	err = service.Run(service.Modules{t, w})
	if err != nil {
		fmt.Println(err)
	}
	// Output: 1 error occurred:
	//	* failed to run module watcher.Watcher: watcher error
}
