package memory_test

import (
	"testing"

	"github.com/elisasre/go-common/v2/auth/cache/cachetest"
	"github.com/elisasre/go-common/v2/auth/store/memory"
)

func TestRotateKeys(t *testing.T) {
	cachetest.RunSuite(t, memory.New())
}
