package netutil_test

import (
	"fmt"
	"net"
	"testing"

	"github.com/elisasre/go-common/v2/netutil"
	"github.com/stretchr/testify/require"
)

func TestGetFreeLocalhostTCPPort(t *testing.T) {
	port, err := netutil.GetFreeLocalhostTCPPort()
	require.NoError(t, err)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	require.NoError(t, listener.Close())
}
